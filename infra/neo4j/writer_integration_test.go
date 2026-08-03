//go:build integration

package neo4j

import (
	"context"
	"testing"

	"github.com/BMokarzel/rhaxis-code.git/domain"
)

func TestWriter_UpsertNode_RoundTrip(t *testing.T) {
	r, drv, teardown := setup(t)
	defer teardown()
	w := NewWriter(drv, "")

	// Upsert um Service novo, depois lê via Reader e confere.
	svc := &domain.Service{
		BaseNode: domain.BaseNode{
			URNValue:      "urn:cg:notifications:_:service",
			KindValue:     domain.KindService,
			NameValue:     "notifications",
			LanguageValue: "ts",
			ResolvedValue: true,
		},
		Framework: "nestjs",
	}
	if err := w.UpsertNode(context.Background(), svc); err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	sm, err := r.LoadServiceMap(context.Background(), nil)
	if err != nil {
		t.Fatalf("LoadServiceMap: %v", err)
	}
	// fixture tinha 2 serviços; agora deve ter 3.
	if len(sm.Services) != 3 {
		t.Errorf("expected 3 services after upsert, got %d", len(sm.Services))
	}
	// Idempotência: mesmo upsert, mesma contagem.
	if err := w.UpsertNode(context.Background(), svc); err != nil {
		t.Fatalf("re-UpsertNode: %v", err)
	}
	sm2, _ := r.LoadServiceMap(context.Background(), nil)
	if len(sm2.Services) != 3 {
		t.Errorf("expected still 3 services after re-upsert, got %d", len(sm2.Services))
	}
}

func TestWriter_ResolveCallHTTPTarget_LinkerHappyPath(t *testing.T) {
	r, drv, teardown := setup(t)
	defer teardown()
	w := NewWriter(drv, "")

	// A fixture tem uma CallHTTP para users-api /notify (unresolved) e
	// nenhum endpoint alvo. Cria o endpoint alvo e roda o linker.
	notifyEndpoint := &domain.Endpoint{
		BaseNode: domain.BaseNode{
			URNValue:        "urn:cg:users-api:ts:endpoint:POST /notify",
			KindValue:       domain.KindEndpoint,
			NameValue:       "notify",
			ServiceURNValue: "urn:cg:users-api:_:service",
			LanguageValue:   "ts",
			ResolvedValue:   true,
		},
		HTTPMethod:   "POST",
		PathTemplate: "/notify",
	}
	if err := w.UpsertNode(context.Background(), notifyEndpoint); err != nil {
		t.Fatalf("UpsertNode(endpoint): %v", err)
	}
	// Também precisa da edge :EXPOSES do users-api para o endpoint (senão
	// nada muda estruturalmente — mas o linker não depende disso).
	if err := w.UpsertEdge(context.Background(),
		"urn:cg:users-api:_:service",
		"urn:cg:users-api:ts:endpoint:POST /notify",
		domain.EdgeExposes, nil); err != nil {
		t.Fatalf("UpsertEdge: %v", err)
	}

	callURN := domain.URN("urn:cg:orders-api:ts:callHTTP:endpoints/post-orders#1/then/1")
	targetURN := domain.URN("urn:cg:users-api:ts:endpoint:POST /notify")

	if err := w.ResolveCallHTTPTarget(context.Background(), callURN, targetURN); err != nil {
		t.Fatalf("ResolveCallHTTPTarget: %v", err)
	}

	// Confere: o slot de expansão da CallHTTP agora deve estar resolvido.
	flow, err := r.LoadEndpointFlow(context.Background(), "urn:cg:orders-api:ts:endpoint:POST /orders")
	if err != nil {
		t.Fatalf("LoadEndpointFlow: %v", err)
	}
	ifNode := flow.Root.Children[1]
	var call *domain.FlowNode
	for i, c := range ifNode.Branches["then"] {
		if c.Node.Kind() == domain.KindCallHTTP {
			call = &ifNode.Branches["then"][i]
		}
	}
	if call == nil {
		t.Fatalf("no CallHTTP found in 'then' branch")
	}
	if !call.Node.Resolved() {
		t.Errorf("expected CallHTTP.Resolved()=true after linker")
	}
	if call.Expansion == nil || call.Expansion.TargetURN != targetURN {
		t.Errorf("expected expansion pointing at %s, got %+v", targetURN, call.Expansion)
	}

	// E o :DEPENDS_ON deve ter ganho weight.
	sm, err := r.LoadServiceMap(context.Background(), nil)
	if err != nil {
		t.Fatalf("LoadServiceMap: %v", err)
	}
	var found bool
	for _, d := range sm.Dependencies {
		if d.From == "urn:cg:orders-api:_:service" && d.To == "urn:cg:users-api:_:service" && d.Via == "http" {
			found = true
			if d.Weight < 2 {
				t.Errorf("expected weight >= 2 (fixture had 1, linker adds 1), got %d", d.Weight)
			}
		}
	}
	if !found {
		t.Errorf("DEPENDS_ON orders-api -> users-api not found")
	}
}

func TestWriter_DeleteNode_Idempotent(t *testing.T) {
	_, drv, teardown := setup(t)
	defer teardown()
	w := NewWriter(drv, "")

	// URN que não existe: não deve erro.
	if err := w.DeleteNode(context.Background(), "urn:cg:does-not-exist"); err != nil {
		t.Errorf("DeleteNode should be idempotent for missing urn: %v", err)
	}
}
