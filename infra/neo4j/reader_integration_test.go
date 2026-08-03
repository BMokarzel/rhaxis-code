//go:build integration

// Testes ponta-a-ponta contra Neo4j real (via testcontainers).
//
// Rodar com:  go test -tags=integration ./infra/neo4j/...
//
// Requer Docker rodando. Se não estiver, o teste faz Skip em vez de falhar,
// para não prejudicar CI sem docker.
package neo4j

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/BMokarzel/rhaxis-code.git/domain"
	"github.com/BMokarzel/rhaxis-code.git/repository"
	driver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	tcneo4j "github.com/testcontainers/testcontainers-go/modules/neo4j"
)

const testPassword = "testtest01"

func setup(t *testing.T) (*Reader, driver.DriverWithContext, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)

	container, err := tcneo4j.Run(ctx, "neo4j:5",
		tcneo4j.WithAdminPassword(testPassword),
	)
	if err != nil {
		cancel()
		t.Skipf("skipping: cannot start neo4j container (docker unavailable?): %v", err)
	}
	uri, err := container.BoltUrl(ctx)
	if err != nil {
		container.Terminate(ctx)
		cancel()
		t.Fatalf("get bolt url: %v", err)
	}

	drv, err := Open(ctx, Config{URI: uri, Username: "neo4j", Password: testPassword})
	if err != nil {
		container.Terminate(ctx)
		cancel()
		t.Fatalf("open driver: %v", err)
	}

	// Aplica schema + fixture.
	runCypherFile(t, ctx, drv, schemaPath(t))
	runCypherFile(t, ctx, drv, fixturePath(t))

	teardown := func() {
		drv.Close(ctx)
		container.Terminate(ctx)
		cancel()
	}
	return NewReader(drv, ""), drv, teardown
}

func schemaPath(t *testing.T) string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "schema", "schema.cypher")
}

func fixturePath(t *testing.T) string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "schema", "fixtures", "nestjs-orders.cypher")
}

// runCypherFile executa um arquivo .cypher separando statements por ";".
// Comentários "//" são strippados linha-a-linha antes do split.
func runCypherFile(t *testing.T, ctx context.Context, drv driver.DriverWithContext, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cleaned := stripComments(string(raw))
	stmts := splitStatements(cleaned)

	session := drv.NewSession(ctx, driver.SessionConfig{AccessMode: driver.AccessModeWrite})
	defer session.Close(ctx)
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := session.Run(ctx, stmt, nil); err != nil {
			t.Fatalf("cypher failed [%s]: %v\n---\n%s", filepath.Base(path), err, stmt)
		}
	}
}

var lineComment = regexp.MustCompile(`(?m)//.*$`)

func stripComments(s string) string {
	return lineComment.ReplaceAllString(s, "")
}

func splitStatements(s string) []string {
	// Split simples por ";". Nossa fixture não usa ";" dentro de strings.
	return strings.Split(s, ";")
}

// --- tests ---------------------------------------------------------------

func TestReader_LoadServiceMap(t *testing.T) {
	r, _, teardown := setup(t)
	defer teardown()

	sm, err := r.LoadServiceMap(context.Background(), nil)
	if err != nil {
		t.Fatalf("LoadServiceMap: %v", err)
	}
	if len(sm.Services) != 2 {
		t.Errorf("expected 2 services, got %d: %+v", len(sm.Services), sm.Services)
	}
	if len(sm.ExternalSystems) != 1 {
		t.Errorf("expected 1 external system (postgres), got %d", len(sm.ExternalSystems))
	}
	if len(sm.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(sm.Dependencies))
	}
	if len(sm.Dependencies) > 0 {
		d := sm.Dependencies[0]
		if d.Via != "http" || d.From == "" || d.To == "" {
			t.Errorf("unexpected dep: %+v", d)
		}
	}
}

func TestReader_ListEndpoints(t *testing.T) {
	r, _, teardown := setup(t)
	defer teardown()

	el, err := r.ListEndpoints(context.Background(), "urn:cg:orders-api:_:service")
	if err != nil {
		t.Fatalf("ListEndpoints: %v", err)
	}
	if el.Service.Name() != "orders-api" {
		t.Errorf("wrong service: %s", el.Service.Name())
	}
	if len(el.Endpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(el.Endpoints))
	}
	// Ordem estável: GET antes de POST (alfabético do método).
	if el.Endpoints[0].HTTPMethod != "GET" || el.Endpoints[1].HTTPMethod != "POST" {
		t.Errorf("unexpected order: %+v", el.Endpoints)
	}
}

func TestReader_LoadEndpointFlow_POSTOrders(t *testing.T) {
	r, _, teardown := setup(t)
	defer teardown()

	fl, err := r.LoadEndpointFlow(context.Background(), "urn:cg:orders-api:ts:endpoint:POST /orders")
	if err != nil {
		t.Fatalf("LoadEndpointFlow: %v", err)
	}
	if fl.Endpoint.PathTemplate != "/orders" {
		t.Errorf("wrong endpoint: %+v", fl.Endpoint)
	}
	if len(fl.Root.Children) != 3 {
		t.Fatalf("expected 3 root steps (auth, if, serialize), got %d", len(fl.Root.Children))
	}
	// step 0: CallFunction auth, expansible.
	if fl.Root.Children[0].Node.Kind() != domain.KindCallFunction {
		t.Errorf("step 0 not CallFunction: %s", fl.Root.Children[0].Node.Kind())
	}
	if fl.Root.Children[0].Expansion == nil {
		t.Errorf("step 0 should have expansion slot")
	}
	// step 1: IfNode com then/else.
	ifNode := fl.Root.Children[1]
	if ifNode.Node.Kind() != domain.KindIfNode {
		t.Errorf("step 1 not IfNode: %s", ifNode.Node.Kind())
	}
	if len(ifNode.Branches) != 2 {
		t.Errorf("expected 2 branches, got %d: %+v", len(ifNode.Branches), ifNode.Branches)
	}
	if _, ok := ifNode.Branches["then"]; !ok {
		t.Errorf("missing 'then' branch")
	}
	// then tem [callDB, callHTTP-unresolved]
	thenChildren := ifNode.Branches["then"]
	if len(thenChildren) != 2 {
		t.Fatalf("expected 2 'then' children, got %d", len(thenChildren))
	}
	// Encontrar a callHTTP não-resolvida.
	var unresolved *domain.FlowNode
	for i := range thenChildren {
		if thenChildren[i].Node.Kind() == domain.KindCallHTTP {
			unresolved = &thenChildren[i]
		}
	}
	if unresolved == nil {
		t.Fatalf("no CallHTTP in 'then' branch")
	}
	if unresolved.Node.Resolved() {
		t.Errorf("expected CallHTTP.resolved=false")
	}
}

func TestReader_ExpandFlow_Function(t *testing.T) {
	r, _, teardown := setup(t)
	defer teardown()

	// Expandir a Function do middleware — deve trazer 1 Block child.
	fn, err := r.ExpandFlow(context.Background(),
		"urn:cg:orders-api:ts:function:src/middleware/auth.ts#authMiddleware")
	if err != nil {
		t.Fatalf("ExpandFlow: %v", err)
	}
	if fn.Node.Kind() != domain.KindFunction {
		t.Errorf("expected Function, got %s", fn.Node.Kind())
	}
	if len(fn.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(fn.Children))
	}
}

func TestReader_ExpandFlow_UnresolvedTarget(t *testing.T) {
	r, _, teardown := setup(t)
	defer teardown()

	// URN que não existe no grafo — Reader deve devolver stub Resolved()=false,
	// não erro.
	fn, err := r.ExpandFlow(context.Background(),
		"urn:cg:_external:http:POST /notify@users-api")
	if err != nil {
		t.Fatalf("ExpandFlow: %v", err)
	}
	if fn.Node.Resolved() {
		t.Errorf("expected stub with Resolved()=false")
	}
}

// compile-time proof of interface conformance
var _ repository.Reader = (*Reader)(nil)
