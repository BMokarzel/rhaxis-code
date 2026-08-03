package domain

import "testing"

func TestDefaultRegistry_HasAllV1Kinds(t *testing.T) {
	expected := []Kind{
		KindService, KindDatabase, KindBroker,
		KindEndpoint, KindFunction, KindMethod,
		KindStruct, KindInterface, KindType,
		KindIfNode, KindSwitchNode, KindLoopNode, KindTryNode, KindBlock,
		KindCallFunction, KindCallHTTP, KindCallDB,
		KindPublish, KindConsume,
	}
	for _, k := range expected {
		if _, ok := DefaultRegistry.Get(k); !ok {
			t.Errorf("kind %q not registered", k)
		}
	}
}

func TestDecodeEndpoint_HappyPath(t *testing.T) {
	props := map[string]any{
		"urn":          "urn:cg:orders:ts:endpoint:POST /orders",
		"kind":         "Endpoint",
		"name":         "createOrder",
		"serviceURN":   "urn:cg:orders:_:service",
		"language":     "ts",
		"resolved":     true,
		"httpMethod":   "POST",
		"pathTemplate": "/orders",
		"framework":    "nestjs",
		"handlerURN":   "urn:cg:orders:ts:method:src/orders.controller.ts#OrdersController.create",
	}
	n, err := DecodeByKind(KindEndpoint, props)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	ep, ok := n.(*Endpoint)
	if !ok {
		t.Fatalf("expected *Endpoint, got %T", n)
	}
	if ep.HTTPMethod != "POST" || ep.PathTemplate != "/orders" {
		t.Errorf("wrong endpoint props: %+v", ep)
	}
	if ep.URN() != URN("urn:cg:orders:ts:endpoint:POST /orders") {
		t.Errorf("wrong URN: %s", ep.URN())
	}
	if !ep.Resolved() {
		t.Errorf("expected resolved=true")
	}
}

func TestDecodeCallHTTP_UnresolvedTarget(t *testing.T) {
	props := map[string]any{
		"urn":          "urn:cg:orders:ts:callHttp:src/x.ts#f@0",
		"kind":         "CallHTTP",
		"httpMethod":   "GET",
		"pathTemplate": "/users/:id",
		"targetHint":   "USERS_API_URL",
		"resolved":     false,
		// targetURN ausente => alvo ainda não linkado
	}
	n, err := DecodeByKind(KindCallHTTP, props)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	call := n.(*CallHTTP)
	if call.TargetURN != nil {
		t.Errorf("expected nil TargetURN, got %v", *call.TargetURN)
	}
	if call.Resolved() {
		t.Errorf("expected resolved=false")
	}
}

func TestDecoder_MissingRequiredKind_Errors(t *testing.T) {
	_, err := DecodeByKind(KindEndpoint, map[string]any{"urn": "x"}) // sem kind prop
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExpansionRule_CallFunctionIsExpandable(t *testing.T) {
	d, _ := DefaultRegistry.Get(KindCallFunction)
	if !d.IsExpandable() {
		t.Errorf("CallFunction should be expandable")
	}
	if d.Expansion.FollowEdge != EdgeExpandsTo {
		t.Errorf("wrong follow edge: %s", d.Expansion.FollowEdge)
	}
}

func TestExpansionRule_FunctionIsNotExpandable(t *testing.T) {
	d, _ := DefaultRegistry.Get(KindFunction)
	if d.IsExpandable() {
		t.Errorf("Function itself should NOT be expandable (só Calls são)")
	}
}

func TestDisplayHints_TableByKind(t *testing.T) {
	// Amostra cobrindo cada categoria — verifica que shape/category das
	// funções derivadoras (usadas pela view HTTP) não regridem silenciosamente.
	// Nada é escrito no grafo: essas funções são consultadas em request time.
	cases := map[Kind]struct{ shape, category string }{
		KindEntry:        {"circle", "start"},
		KindExit:         {"circle", "end"},
		KindJoin:         {"circle", "control"},
		KindIfNode:       {"diamond", "control"},
		KindTryNode:      {"hexagon", "control"},
		KindReturn:       {"triangle", "terminal"},
		KindCallHTTP:     {"rect", "io-http"},
		KindCallDB:       {"rect", "io-db"},
		KindLogNode:      {"rect", "log"},
		KindEndpoint:     {"portal", "entrypoint"},
		KindService:      {"capsule", "aggregator"},
		KindDatabase:     {"cylinder", "infrastructure"},
		KindMiddleware:   {"shield", "middleware"},
		KindErrorType:    {"shield", "error"},
		KindConfig:       {"note", "config"},
		KindLanguage:     {"chip", "tech"},
		KindMethod:       {"rect", "container"},
		KindStruct:       {"rect", "type"},
		KindPublish:      {"parallelogram", "io-broker"},
		KindCallFunction: {"rect", "call"},
	}
	for kind, want := range cases {
		if got := DisplayShapeFor(kind); got != want.shape {
			t.Errorf("kind %s: DisplayShapeFor = %q, want %q", kind, got, want.shape)
		}
		if got := DisplayCategoryFor(kind); got != want.category {
			t.Errorf("kind %s: DisplayCategoryFor = %q, want %q", kind, got, want.category)
		}
	}
}

func TestDisplayHints_EveryRegisteredKindCovered(t *testing.T) {
	// Guard: se um kind for adicionado ao registry mas esquecido na tabela
	// de DisplayShapeFor/DisplayCategoryFor, este teste falha e força atualização.
	for _, d := range DefaultRegistry.All() {
		if DisplayShapeFor(d.Kind) == "" {
			t.Errorf("kind %s missing DisplayShapeFor entry", d.Kind)
		}
		if DisplayCategoryFor(d.Kind) == "" {
			t.Errorf("kind %s missing DisplayCategoryFor entry", d.Kind)
		}
	}
}

func TestEncoder_DoesNotPersistDisplayHints(t *testing.T) {
	// Regressão: display hints são derivados pela view HTTP, jamais persistidos.
	// Se voltarem para encodeBase, dados no Neo4j crescem em N nodes × 2 props
	// e mudança em displayShape exige rebuild do grafo.
	for _, d := range DefaultRegistry.All() {
		base := BaseNode{URNValue: "urn:test", KindValue: d.Kind, NameValue: "x"}
		props := encodeBase(base)
		if _, ok := props["displayShape"]; ok {
			t.Errorf("kind %s: encodeBase should not write displayShape", d.Kind)
		}
		if _, ok := props["displayCategory"]; ok {
			t.Errorf("kind %s: encodeBase should not write displayCategory", d.Kind)
		}
	}
}
