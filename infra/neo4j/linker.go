package neo4j

import (
	"context"
	"fmt"
	"strings"

	"github.com/BMokarzel/rhaxis-code.git/domain"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Linker resolve CallHTTP ainda não linkadas contra Endpoints no grafo,
// matchando por {httpMethod, pathTemplate normalizado}. Só resolve quando
// o match é único — múltiplos candidatos ficam para revisão humana.
//
// Trade-offs:
//   - Match por rota exata (após normalização) evita falsos positivos comuns
//     de match por targetHint (que é heurístico: nome de env var etc).
//   - Não usa targetHint no v1: teria que carregar Config nodes e resolver
//     host, o que expande o escopo. Fica para v2.
//   - Ambiguidade (2+ endpoints com mesma rota em serviços diferentes) é
//     LinkResult.Ambiguous — o CLI só reporta e não resolve.
type Linker struct {
	drv      neo4j.DriverWithContext
	database string
	writer   *Writer
}

// NewLinker constrói um Linker sobre o mesmo driver que o Writer.
func NewLinker(drv neo4j.DriverWithContext, database string) *Linker {
	return &Linker{
		drv:      drv,
		database: database,
		writer:   NewWriter(drv, database),
	}
}

// LinkStats sumariza o resultado de Link — útil para logs e testes.
type LinkStats struct {
	CallsScanned   int // total de CallHTTP unresolved encontradas
	Resolved       int // linked com sucesso (match único)
	Ambiguous      int // 2+ endpoints candidatos, sem escolha
	NoMatch        int // nenhum endpoint com rota compatível
	AlreadyLinked  int // eram unresolved mas outro passe já linkou
	SelfLoopSkip   int // match aponta pro próprio serviço da chamada
	DependsUpdated int // Service→Service DEPENDS_ON criadas/incrementadas
}

// Ambiguity é reportado ao chamador quando existem múltiplos candidatos
// pra mesma CallHTTP — o linker se recusa a escolher.
type Ambiguity struct {
	CallURN     domain.URN
	Method      string
	PathPattern string
	Candidates  []domain.URN
}

// EnsureRouteIndex cria (idempotente) o índice composto que acelera o match
// {httpMethod, pathTemplate}. Sem ele, cada Link vira scan cheio de Endpoints.
// Cypher CREATE INDEX IF NOT EXISTS é seguro rodar em cada invocação.
func (l *Linker) EnsureRouteIndex(ctx context.Context) error {
	const q = `CREATE INDEX endpoint_route IF NOT EXISTS FOR (e:Endpoint) ON (e.httpMethod, e.pathTemplate)`
	session := l.drv.NewSession(ctx, writeSessionConfig(l.database))
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, q, nil)
	})
	if err != nil {
		return fmt.Errorf("neo4j: EnsureRouteIndex: %w", err)
	}
	return nil
}

// unresolvedCall é o shape lido da query — apenas o que o matcher precisa.
type unresolvedCall struct {
	URN         domain.URN
	Method      string
	PathPattern string // normalizado
}

// candidateEndpoint é o shape dos possíveis alvos.
type candidateEndpoint struct {
	URN         domain.URN
	PathPattern string // normalizado
}

const cypherListUnresolvedCallHTTP = `
MATCH (c:CallHTTP)
WHERE (c.resolved = false OR c.resolved IS NULL)
  AND c.httpMethod IS NOT NULL
  AND c.pathTemplate IS NOT NULL
RETURN c.urn AS urn, c.httpMethod AS method, c.pathTemplate AS path
`

// cypherListEndpointsForMethod devolve todos os endpoints de um dado método.
// Filtro por rota exata acontece em Go (após normalizar dos dois lados) —
// mais robusto que tentar Cypher-side matching com estilos mistos (:id, {id}).
const cypherListEndpointsForMethod = `
MATCH (e:Endpoint {httpMethod: $method})
WHERE e.pathTemplate IS NOT NULL
RETURN e.urn AS urn, e.pathTemplate AS path
`

// Link executa uma passada de linking sobre TODAS as CallHTTP não resolvidas.
// Idempotente: rodar de novo só reprocessa o que ainda estiver unresolved.
// Retorna stats + lista de ambiguidades pra reporting.
func (l *Linker) Link(ctx context.Context) (LinkStats, []Ambiguity, error) {
	var stats LinkStats
	var ambiguous []Ambiguity

	if err := l.EnsureRouteIndex(ctx); err != nil {
		return stats, nil, err
	}

	calls, err := l.listUnresolved(ctx)
	if err != nil {
		return stats, nil, err
	}
	stats.CallsScanned = len(calls)

	// Cache de candidatos por método — a maioria dos serviços usa poucos
	// métodos (GET/POST/PUT/DELETE), então cachear economiza queries.
	candidatesByMethod := map[string][]candidateEndpoint{}
	load := func(method string) ([]candidateEndpoint, error) {
		if cs, ok := candidatesByMethod[method]; ok {
			return cs, nil
		}
		cs, err := l.listEndpointsForMethod(ctx, method)
		if err != nil {
			return nil, err
		}
		candidatesByMethod[method] = cs
		return cs, nil
	}

	for _, call := range calls {
		wantPath := normalizePath(call.PathPattern)
		cands, err := load(call.Method)
		if err != nil {
			return stats, ambiguous, err
		}

		var matches []domain.URN
		for _, e := range cands {
			if normalizePath(e.PathPattern) == wantPath {
				matches = append(matches, e.URN)
			}
		}

		switch len(matches) {
		case 0:
			stats.NoMatch++
		case 1:
			target := matches[0]
			// Evita self-loop: chamada dentro do mesmo serviço apontando
			// pro próprio endpoint (loop interno via HTTP). Setamos targetURN
			// mesmo assim, mas não incrementamos DEPENDS_ON (isso é feito
			// pela query — WHERE sFrom <> sTo).
			if serviceSlug(call.URN) == serviceSlug(target) {
				stats.SelfLoopSkip++
			}
			if err := l.writer.ResolveCallHTTPTarget(ctx, call.URN, target); err != nil {
				// Um erro comum é o call já ter sido linkado por outro passe
				// concorrente. Fallback: contabiliza como já-linkado.
				if strings.Contains(err.Error(), "not found") {
					stats.AlreadyLinked++
					continue
				}
				return stats, ambiguous, fmt.Errorf("resolve %s -> %s: %w", call.URN, target, err)
			}
			stats.Resolved++
			if serviceSlug(call.URN) != serviceSlug(target) {
				stats.DependsUpdated++
			}
		default:
			stats.Ambiguous++
			ambiguous = append(ambiguous, Ambiguity{
				CallURN:     call.URN,
				Method:      call.Method,
				PathPattern: wantPath,
				Candidates:  matches,
			})
		}
	}

	return stats, ambiguous, nil
}

func (l *Linker) listUnresolved(ctx context.Context) ([]unresolvedCall, error) {
	session := l.drv.NewSession(ctx, sessionConfig(l.database))
	defer session.Close(ctx)
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypherListUnresolvedCallHTTP, nil)
		if err != nil {
			return nil, err
		}
		return res.Collect(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j: listUnresolvedCallHTTP: %w", err)
	}
	recs := result.([]*neo4j.Record)
	out := make([]unresolvedCall, 0, len(recs))
	for _, r := range recs {
		urn := asStringProp(recordGet(r, "urn"))
		method := asStringProp(recordGet(r, "method"))
		path := asStringProp(recordGet(r, "path"))
		if urn == "" || method == "" || path == "" {
			continue
		}
		out = append(out, unresolvedCall{
			URN:         domain.URN(urn),
			Method:      strings.ToUpper(method),
			PathPattern: path,
		})
	}
	return out, nil
}

func (l *Linker) listEndpointsForMethod(ctx context.Context, method string) ([]candidateEndpoint, error) {
	session := l.drv.NewSession(ctx, sessionConfig(l.database))
	defer session.Close(ctx)
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypherListEndpointsForMethod, map[string]any{
			"method": strings.ToUpper(method),
		})
		if err != nil {
			return nil, err
		}
		return res.Collect(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j: listEndpointsForMethod(%s): %w", method, err)
	}
	recs := result.([]*neo4j.Record)
	out := make([]candidateEndpoint, 0, len(recs))
	for _, r := range recs {
		urn := asStringProp(recordGet(r, "urn"))
		path := asStringProp(recordGet(r, "path"))
		if urn == "" || path == "" {
			continue
		}
		out = append(out, candidateEndpoint{
			URN:         domain.URN(urn),
			PathPattern: path,
		})
	}
	return out, nil
}

// normalizePath canonicaliza um path template pra que match funcione entre
// diferentes convenções (:id, {id}, <id>). Também colapsa slashes e força
// leading slash / no trailing slash.
//
// Ex.: "/users/:id/orders/" == "users/{id}/orders" == "/users/<id>/orders"
//      → "/users/:_/orders"
//
// Nota: não descartamos o nome do param (:_ como marcador anônimo) porque
// dois endpoints "/users/:id" e "/users/:slug" são a MESMA rota HTTP — só
// difere o nome interno da variável, irrelevante pro linker.
func normalizePath(p string) string {
	if p == "" {
		return ""
	}
	// Força leading /
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// Remove trailing / (exceto se for root)
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimRight(p, "/")
	}
	// Substitui placeholders por :_ anônimo.
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if s == "" {
			continue
		}
		if strings.HasPrefix(s, ":") {
			segs[i] = ":_"
			continue
		}
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			segs[i] = ":_"
			continue
		}
		if strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">") {
			segs[i] = ":_"
			continue
		}
	}
	return strings.Join(segs, "/")
}

// serviceSlug extrai o slug de serviço do URN. Formato canônico:
// urn:cg:<slug>:<lang>:<kind>:<...>. Devolve "" se malformado.
func serviceSlug(u domain.URN) string {
	parts := strings.SplitN(string(u), ":", 4)
	if len(parts) < 3 || parts[0] != "urn" || parts[1] != "cg" {
		return ""
	}
	return parts[2]
}
