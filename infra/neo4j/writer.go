package neo4j

import (
	"context"
	"fmt"
	"strings"

	"github.com/BMokarzel/rhaxis-code.git/domain"
	"github.com/BMokarzel/rhaxis-code.git/repository"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)


// Writer implementa repository.Writer contra Neo4j 5.x.
type Writer struct {
	drv      neo4j.DriverWithContext
	database string
}

// NewWriter constrói um Writer a partir de um driver já aberto.
func NewWriter(drv neo4j.DriverWithContext, database string) *Writer {
	return &Writer{drv: drv, database: database}
}

var _ repository.Writer = (*Writer)(nil)

func writeSessionConfig(database string) neo4j.SessionConfig {
	cfg := neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite}
	if database != "" {
		cfg.DatabaseName = database
	}
	return cfg
}

// -------- UpsertNode ---------------------------------------------------

// upsertNodeCypher é construído dinamicamente porque labels dependem do kind.
// Formato: MERGE (n:Node:<Label> {urn:$urn}) SET n = $props, n:Node:<Label>
// Nota: SET n = $props zera props antigas e reaplica — comportamento
// esperado por "upsert idempotente".
func (w *Writer) UpsertNode(ctx context.Context, n domain.Node) error {
	extraLabels, props, err := domain.EncodeNode(n)
	if err != nil {
		return fmt.Errorf("neo4j: encode node %s: %w", n.URN(), err)
	}
	var lb strings.Builder
	lb.WriteString(":Node")
	for _, l := range extraLabels {
		if err := validateIdent(l); err != nil {
			return fmt.Errorf("neo4j: invalid label %q: %w", l, err)
		}
		lb.WriteByte(':')
		lb.WriteString(l)
	}
	cypher := fmt.Sprintf(`MERGE (n%s {urn: $urn}) SET n = $props`, lb.String())

	session := w.drv.NewSession(ctx, writeSessionConfig(w.database))
	defer session.Close(ctx)
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, cypher, map[string]any{
			"urn":   string(n.URN()),
			"props": props,
		})
	})
	if err != nil {
		return fmt.Errorf("neo4j: UpsertNode(%s): %w", n.URN(), err)
	}
	return nil
}

// -------- UpsertEdge ---------------------------------------------------

func (w *Writer) UpsertEdge(ctx context.Context, from, to domain.URN, kind domain.EdgeType, props map[string]any) error {
	if err := validateIdent(string(kind)); err != nil {
		return fmt.Errorf("neo4j: invalid edge type %q: %w", kind, err)
	}
	// Requer que ambos os endpoints existam (MATCH, não MERGE).
	// Props são atualizadas via SET r += $props (não zera props que já existem
	// e não foram passadas — comportamento apropriado para edges com metadata
	// posicional como :CONTAINS {index}).
	cypher := fmt.Sprintf(`
MATCH (a {urn: $from}), (b {urn: $to})
MERGE (a)-[r:%s]->(b)
SET r += $props
RETURN r`, string(kind))

	session := w.drv.NewSession(ctx, writeSessionConfig(w.database))
	defer session.Close(ctx)
	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypher, map[string]any{
			"from":  string(from),
			"to":    string(to),
			"props": normalizeProps(props),
		})
		if err != nil {
			return nil, err
		}
		return res.Collect(ctx)
	})
	if err != nil {
		return fmt.Errorf("neo4j: UpsertEdge(%s->%s :%s): %w", from, to, kind, err)
	}
	if len(result.([]*neo4j.Record)) == 0 {
		return fmt.Errorf("neo4j: UpsertEdge: one or both endpoints not found (from=%s, to=%s)", from, to)
	}
	return nil
}

// -------- DeleteNode ---------------------------------------------------

func (w *Writer) DeleteNode(ctx context.Context, urn domain.URN) error {
	cypher := `MATCH (n {urn: $urn}) DETACH DELETE n`
	session := w.drv.NewSession(ctx, writeSessionConfig(w.database))
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(ctx, cypher, map[string]any{"urn": string(urn)})
	})
	if err != nil {
		return fmt.Errorf("neo4j: DeleteNode(%s): %w", urn, err)
	}
	return nil
}

// -------- ResolveCallHTTPTarget ---------------------------------------

// Operação atômica: em uma tx, verifica tipos, seta props, cria :EXPANDS_TO,
// e materializa :DEPENDS_ON (via='http', weight incrementado).
//
// Design notes:
//   - EXPANDS_TO em vez de CALLS: v1 mantém CALLS estrito a chamadas
//     intra-processo. Cross-service usa apenas a affordance EXPANDS_TO —
//     UI expande, análise topológica usa DEPENDS_ON.
//   - serviceURN dos nodes não é sempre persistido pelo loader (herda
//     implicitamente pelo prefixo do URN). Derivamos aqui via
//     split(urn,':')[2] pra funcionar com payloads reais E com fixtures
//     que trazem a prop explícita.
//   - Preferimos serviceURN da prop se presente (fixture), caindo pra
//     derivação caso null.
const cypherResolveCallHTTP = `
MATCH (call:CallHTTP {urn: $callURN})
MATCH (target:Endpoint {urn: $targetURN})
SET call.targetURN = $targetURN, call.resolved = true
MERGE (call)-[:EXPANDS_TO]->(target)

WITH call, target,
     coalesce(call.serviceURN,   'urn:cg:' + split(call.urn,   ':')[2] + ':_:service') AS fromSvcURN,
     coalesce(target.serviceURN, 'urn:cg:' + split(target.urn, ':')[2] + ':_:service') AS toSvcURN
MATCH (sFrom:Service {urn: fromSvcURN})
MATCH (sTo:Service {urn: toSvcURN})
WHERE sFrom <> sTo
MERGE (sFrom)-[d:DEPENDS_ON {via:'http'}]->(sTo)
  ON CREATE SET d.weight = 1
  ON MATCH  SET d.weight = coalesce(d.weight, 0) + 1
RETURN call.urn AS callURN, target.urn AS targetURN
`

func (w *Writer) ResolveCallHTTPTarget(ctx context.Context, callURN, targetEndpointURN domain.URN) error {
	session := w.drv.NewSession(ctx, writeSessionConfig(w.database))
	defer session.Close(ctx)
	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypherResolveCallHTTP, map[string]any{
			"callURN":   string(callURN),
			"targetURN": string(targetEndpointURN),
		})
		if err != nil {
			return nil, err
		}
		return res.Collect(ctx)
	})
	if err != nil {
		return fmt.Errorf("neo4j: ResolveCallHTTPTarget(%s -> %s): %w",
			callURN, targetEndpointURN, err)
	}
	recs := result.([]*neo4j.Record)
	if len(recs) == 0 {
		return fmt.Errorf("neo4j: ResolveCallHTTPTarget: call or target not found (call=%s, target=%s) — either does not exist or wrong kind",
			callURN, targetEndpointURN)
	}
	return nil
}

// -------- helpers ------------------------------------------------------

// validateIdent impede injeção via label/relationship type. Aceita apenas
// identificadores alfanuméricos (+ underscore), começando por letra.
func validateIdent(s string) error {
	if s == "" {
		return fmt.Errorf("empty")
	}
	if !isLetter(s[0]) {
		return fmt.Errorf("must start with a letter")
	}
	for _, r := range s {
		if !(isLetter(byte(r)) || (r >= '0' && r <= '9') || r == '_') {
			return fmt.Errorf("contains invalid char %q", r)
		}
	}
	return nil
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// normalizeProps trata nil map (driver aceita, mas é feio) e strings vazias.
func normalizeProps(p map[string]any) map[string]any {
	if p == nil {
		return map[string]any{}
	}
	return p
}

