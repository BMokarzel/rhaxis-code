package loader

import (
	"context"
	"strings"
	"testing"

	"github.com/BMokarzel/rhaxis-code.git/domain"
)

// fakeWriter é um Writer in-memory. Guarda apenas o suficiente para
// validar que o loader chamou o que devia.
type fakeWriter struct {
	nodes map[domain.URN]domain.Node
	edges []fakeEdge
}

type fakeEdge struct {
	from  domain.URN
	to    domain.URN
	kind  domain.EdgeType
	props map[string]any
}

func newFakeWriter() *fakeWriter {
	return &fakeWriter{nodes: map[domain.URN]domain.Node{}}
}

func (f *fakeWriter) UpsertNode(_ context.Context, n domain.Node) error {
	f.nodes[n.URN()] = n
	return nil
}
func (f *fakeWriter) UpsertEdge(_ context.Context, from, to domain.URN, k domain.EdgeType, p map[string]any) error {
	f.edges = append(f.edges, fakeEdge{from, to, k, p})
	return nil
}
func (f *fakeWriter) DeleteNode(_ context.Context, _ domain.URN) error { return nil }
func (f *fakeWriter) ResolveCallHTTPTarget(_ context.Context, _, _ domain.URN) error {
	return nil
}

// samplePayload é o mesmo que o extrator produz para fixtures/sample-nestjs.
// Mantemos inline (não lemos arquivo) para o teste ser hermético.
const samplePayload = `{
  "schemaVersion": "1",
  "extractedAt": "2026-08-01T15:41:07.452Z",
  "service": {
    "urn": "urn:cg:sample-orders:_:service",
    "name": "sample-nestjs",
    "language": "ts",
    "framework": "nestjs",
    "runtime": "node"
  },
  "nodes": [
    { "urn": "urn:cg:sample-orders:_:service", "kind": "Service", "name": "sample-nestjs",
      "properties": { "framework": "nestjs", "runtime": "node" } },
    { "urn": "urn:cg:sample-orders:ts:endpoint:POST /orders", "kind": "Endpoint",
      "name": "POST /orders",
      "properties": { "httpMethod": "POST", "pathTemplate": "/orders", "framework": "nestjs" } },
    { "urn": "urn:cg:sample-orders:ts:endpoint:POST /orders/if@0", "kind": "IfNode",
      "name": "if body.notify",
      "properties": { "conditionText": "body.notify" } },
    { "urn": "urn:cg:sample-orders:ts:endpoint:POST /orders/if@0/then/callHttp@0", "kind": "CallHTTP",
      "name": "POST /notify", "resolved": false,
      "properties": { "httpMethod": "POST", "pathTemplate": "/notify", "targetHint": "this.http" } }
  ],
  "edges": [
    { "from": "urn:cg:sample-orders:_:service",
      "to": "urn:cg:sample-orders:ts:endpoint:POST /orders", "type": "EXPOSES" },
    { "from": "urn:cg:sample-orders:ts:endpoint:POST /orders",
      "to": "urn:cg:sample-orders:ts:endpoint:POST /orders/if@0",
      "type": "CONTAINS", "props": { "index": 0 } }
  ]
}`

func TestLoadInto_HappyPath(t *testing.T) {
	p, err := ReadPayload(strings.NewReader(samplePayload))
	if err != nil {
		t.Fatalf("ReadPayload: %v", err)
	}

	w := newFakeWriter()
	stats, err := LoadInto(context.Background(), p, w)
	if err != nil {
		t.Fatalf("LoadInto: %v", err)
	}

	if stats.NodesUpserted != 4 {
		t.Errorf("nodes upserted: got %d, want 4", stats.NodesUpserted)
	}
	if stats.EdgesUpserted != 2 {
		t.Errorf("edges upserted: got %d, want 2", stats.EdgesUpserted)
	}
	if stats.NodesSkipped != 0 || stats.EdgesSkipped != 0 {
		t.Errorf("unexpected skips: nodes=%d edges=%d", stats.NodesSkipped, stats.EdgesSkipped)
	}

	// Sanity: o Endpoint node deve estar corretamente decodificado
	// com pathTemplate populado.
	epURN := domain.URN("urn:cg:sample-orders:ts:endpoint:POST /orders")
	ep, ok := w.nodes[epURN].(*domain.Endpoint)
	if !ok {
		t.Fatalf("Endpoint node not found or wrong type: %T", w.nodes[epURN])
	}
	if ep.PathTemplate != "/orders" {
		t.Errorf("pathTemplate: got %q, want /orders", ep.PathTemplate)
	}
	if ep.HTTPMethod != "POST" {
		t.Errorf("httpMethod: got %q, want POST", ep.HTTPMethod)
	}

	// Sanity: o CallHTTP não-resolvido deve chegar com Resolved=false
	// e sem TargetURN.
	callURN := domain.URN("urn:cg:sample-orders:ts:endpoint:POST /orders/if@0/then/callHttp@0")
	call, ok := w.nodes[callURN].(*domain.CallHTTP)
	if !ok {
		t.Fatalf("CallHTTP node not found or wrong type: %T", w.nodes[callURN])
	}
	if call.Resolved() {
		t.Errorf("CallHTTP should be unresolved")
	}
	if call.TargetURN != nil {
		t.Errorf("CallHTTP.TargetURN should be nil, got %v", *call.TargetURN)
	}

	// A edge CONTAINS deve preservar props.index como int64 (via
	// normalizeNumbers), pois o driver Neo4j não aceita json.Number.
	var found bool
	for _, e := range w.edges {
		if e.kind == domain.EdgeContains {
			found = true
			idx, ok := e.props["index"].(int64)
			if !ok {
				t.Fatalf("CONTAINS.props[index] should be int64, got %T", e.props["index"])
			}
			if idx != 0 {
				t.Errorf("CONTAINS.props[index]: got %d, want 0", idx)
			}
		}
	}
	if !found {
		t.Error("no CONTAINS edge captured")
	}
}

func TestReadPayload_RejectsUnknownSchemaVersion(t *testing.T) {
	bad := `{"schemaVersion":"9","service":{"urn":"","name":"","language":""},"nodes":[],"edges":[]}`
	_, err := ReadPayload(strings.NewReader(bad))
	if err == nil {
		t.Fatal("expected error for unknown schemaVersion")
	}
	if !strings.Contains(err.Error(), "schemaVersion") {
		t.Errorf("error should mention schemaVersion, got: %v", err)
	}
}

func TestLoadInto_SkipsUnknownKindsButLoadsRest(t *testing.T) {
	payload := `{
      "schemaVersion":"1","extractedAt":"","service":{"urn":"u","name":"n","language":"ts"},
      "nodes":[
        {"urn":"urn:cg:x:_:service","kind":"Service","name":"x"},
        {"urn":"urn:cg:x:ts:mystery:foo","kind":"UnknownKindV2","name":"foo"}
      ],
      "edges":[]
    }`
	p, err := ReadPayload(strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	w := newFakeWriter()
	stats, err := LoadInto(context.Background(), p, w)
	if err != nil {
		t.Fatalf("LoadInto: %v", err)
	}
	if stats.NodesUpserted != 1 {
		t.Errorf("nodes upserted: got %d, want 1", stats.NodesUpserted)
	}
	if stats.NodesSkipped != 1 {
		t.Errorf("nodes skipped: got %d, want 1", stats.NodesSkipped)
	}
}
