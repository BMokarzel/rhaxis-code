package neo4j

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/BMokarzel/rhaxis-code.git/domain"
	"github.com/BMokarzel/rhaxis-code.git/repository"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Reader implementa repository.Reader contra Neo4j 5.x.
type Reader struct {
	drv      neo4j.DriverWithContext
	database string
}

// NewReader constrói um Reader a partir de um driver já aberto.
// O ciclo de vida do driver (Close) é responsabilidade do chamador.
func NewReader(drv neo4j.DriverWithContext, database string) *Reader {
	return &Reader{drv: drv, database: database}
}

// Compile-time check.
var _ repository.Reader = (*Reader)(nil)

// -------- Tela 1 --------------------------------------------------------

func (r *Reader) LoadServiceMap(ctx context.Context, _ *repository.ServiceMapFilter) (domain.ServiceMap, error) {
	session := r.drv.NewSession(ctx, sessionConfig(r.database))
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypherServiceMap, nil)
		if err != nil {
			return nil, err
		}
		return res.Single(ctx)
	})
	if err != nil {
		return domain.ServiceMap{}, fmt.Errorf("neo4j: LoadServiceMap: %w", err)
	}
	rec := result.(*neo4j.Record)

	out := domain.ServiceMap{}

	// services
	for _, v := range asListProp(recordGet(rec, "services")) {
		if n, ok := asNeoNode(v); ok {
			if dn, ok := nodeToDomainSafe(n); ok {
				if s, ok := dn.(*domain.Service); ok {
					out.Services = append(out.Services, *s)
				}
			}
		}
	}
	// external systems (dbs + brokers)
	for _, key := range []string{"dbs", "brokers"} {
		for _, v := range asListProp(recordGet(rec, key)) {
			if n, ok := asNeoNode(v); ok {
				if dn, ok := nodeToDomainSafe(n); ok {
					out.ExternalSystems = append(out.ExternalSystems, dn)
				}
			}
		}
	}
	// deps (podem vir com from/to nil se o OPTIONAL MATCH não casou)
	for _, v := range asListProp(recordGet(rec, "deps")) {
		m := asMapProp(v)
		if m == nil {
			continue
		}
		from := asStringProp(m["from"])
		to := asStringProp(m["to"])
		if from == "" || to == "" {
			continue
		}
		out.Dependencies = append(out.Dependencies, domain.ServiceDependency{
			From:   domain.URN(from),
			To:     domain.URN(to),
			Via:    asStringProp(m["via"]),
			Weight: asIntProp(m["weight"]),
		})
	}
	return out, nil
}

// -------- Tela 2 --------------------------------------------------------

func (r *Reader) ListEndpoints(ctx context.Context, serviceURN domain.URN) (domain.EndpointList, error) {
	session := r.drv.NewSession(ctx, sessionConfig(r.database))
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypherListEndpoints, map[string]any{"serviceURN": string(serviceURN)})
		if err != nil {
			return nil, err
		}
		return res.Single(ctx)
	})
	if err != nil {
		return domain.EndpointList{}, fmt.Errorf("neo4j: ListEndpoints(%s): %w", serviceURN, err)
	}
	rec := result.(*neo4j.Record)

	svcNode, ok := asNeoNode(recordGet(rec, "service"))
	if !ok {
		return domain.EndpointList{}, fmt.Errorf("service %q not found", serviceURN)
	}
	svcDomain, err := nodeToDomain(svcNode)
	if err != nil {
		return domain.EndpointList{}, fmt.Errorf("decode service %q: %w", serviceURN, err)
	}
	svc, ok := svcDomain.(*domain.Service)
	if !ok {
		return domain.EndpointList{}, fmt.Errorf("node %q is not a Service (kind=%s)", serviceURN, svcDomain.Kind())
	}

	out := domain.EndpointList{Service: *svc}
	for _, v := range asListProp(recordGet(rec, "endpoints")) {
		n, ok := asNeoNode(v)
		if !ok {
			continue
		}
		dn, ok := nodeToDomainSafe(n)
		if !ok {
			continue
		}
		if ep, ok := dn.(*domain.Endpoint); ok {
			out.Endpoints = append(out.Endpoints, *ep)
		}
	}
	// Ordem estável: por método + path.
	sort.Slice(out.Endpoints, func(i, j int) bool {
		if out.Endpoints[i].HTTPMethod != out.Endpoints[j].HTTPMethod {
			return out.Endpoints[i].HTTPMethod < out.Endpoints[j].HTTPMethod
		}
		return out.Endpoints[i].PathTemplate < out.Endpoints[j].PathTemplate
	})
	return out, nil
}

// -------- Tela 3 --------------------------------------------------------

func (r *Reader) LoadEndpointFlow(ctx context.Context, endpointURN domain.URN) (domain.EndpointFlow, error) {
	session := r.drv.NewSession(ctx, sessionConfig(r.database))
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypherEndpointFlow, map[string]any{"endpointURN": string(endpointURN)})
		if err != nil {
			return nil, err
		}
		return res.Single(ctx)
	})
	if err != nil {
		return domain.EndpointFlow{}, fmt.Errorf("neo4j: LoadEndpointFlow(%s): %w", endpointURN, err)
	}
	rec := result.(*neo4j.Record)

	epNode, ok := asNeoNode(recordGet(rec, "endpoint"))
	if !ok {
		return domain.EndpointFlow{}, fmt.Errorf("endpoint %q not found", endpointURN)
	}
	epDomain, err := nodeToDomain(epNode)
	if err != nil {
		return domain.EndpointFlow{}, fmt.Errorf("decode endpoint %q: %w", endpointURN, err)
	}
	ep, ok := epDomain.(*domain.Endpoint)
	if !ok {
		return domain.EndpointFlow{}, fmt.Errorf("node %q is not an Endpoint (kind=%s)", endpointURN, epDomain.Kind())
	}

	// Root sintético agrega os steps do endpoint.
	root := domain.FlowNode{Node: ep}
	rows := asListProp(recordGet(rec, "rows"))
	root.Children = buildStepChildren(rows)

	return domain.EndpointFlow{Endpoint: *ep, Root: root}, nil
}

// buildStepChildren consome rows ACHATADAS da query da Tela 3 e agrupa em
// FlowNodes. Cada row tem shape:
//
//	{child, idx, callTarget, branchLabel, branchBlock, branchChild, branchIdx, bcCallTarget}
//
// Rows redundantes por produto cartesiano (child × branch × branchChild) são
// deduplicadas por URN. Rows sem branchLabel só contribuem o child raiz.
func buildStepChildren(rows []any) []domain.FlowNode {
	// Estrutura de agrupamento por child.
	type branchKid struct {
		node       domain.Node
		callTarget any
		idx        int
	}
	type childAgg struct {
		fn       domain.FlowNode
		idx      int
		branches map[string][]branchKid
		seen     map[string]map[domain.URN]bool // label -> URN -> present
	}

	byURN := map[domain.URN]*childAgg{}
	order := []domain.URN{} // preserva 1a ocorrência para ordenação estável

	for _, r := range rows {
		m := asMapProp(r)
		if m == nil {
			continue
		}
		childNeo, ok := asNeoNode(m["child"])
		if !ok {
			continue
		}
		childDomain, ok := nodeToDomainSafe(childNeo)
		if !ok {
			continue
		}
		agg, exists := byURN[childDomain.URN()]
		if !exists {
			fn := domain.FlowNode{Node: childDomain}
			if slot := buildExpansionSlot(childDomain, m["callTarget"]); slot != nil {
				fn.Expansion = slot
			}
			agg = &childAgg{
				fn:       fn,
				idx:      asIntProp(m["idx"]),
				branches: map[string][]branchKid{},
				seen:     map[string]map[domain.URN]bool{},
			}
			byURN[childDomain.URN()] = agg
			order = append(order, childDomain.URN())
		}

		// Parte de branch, se existir nesta row.
		label := asStringProp(m["branchLabel"])
		if label == "" {
			continue
		}
		if agg.seen[label] == nil {
			agg.seen[label] = map[domain.URN]bool{}
		}
		bcNeo, ok := asNeoNode(m["branchChild"])
		if !ok {
			// registra que a branch existe mesmo sem child (bloco vazio)
			if _, present := agg.branches[label]; !present {
				agg.branches[label] = nil
			}
			continue
		}
		bcDomain, ok := nodeToDomainSafe(bcNeo)
		if !ok {
			continue
		}
		if agg.seen[label][bcDomain.URN()] {
			continue
		}
		agg.seen[label][bcDomain.URN()] = true
		agg.branches[label] = append(agg.branches[label], branchKid{
			node:       bcDomain,
			callTarget: m["bcCallTarget"],
			idx:        asIntProp(m["branchIdx"]),
		})
	}

	// Materializa Branches em cada FlowNode.
	for _, agg := range byURN {
		if len(agg.branches) == 0 {
			continue
		}
		out := make(map[string][]domain.FlowNode, len(agg.branches))
		for label, kids := range agg.branches {
			sort.SliceStable(kids, func(i, j int) bool { return kids[i].idx < kids[j].idx })
			fns := make([]domain.FlowNode, 0, len(kids))
			for _, k := range kids {
				fn := domain.FlowNode{Node: k.node}
				if slot := buildExpansionSlot(k.node, k.callTarget); slot != nil {
					fn.Expansion = slot
				}
				fns = append(fns, fn)
			}
			out[label] = fns
		}
		agg.fn.Branches = out
	}

	// Ordena children por idx (estável).
	sort.SliceStable(order, func(i, j int) bool {
		return byURN[order[i]].idx < byURN[order[j]].idx
	})
	result := make([]domain.FlowNode, 0, len(order))
	for _, u := range order {
		result = append(result, byURN[u].fn)
	}
	return result
}

// buildExpansionSlot devolve um slot para nodes que a spec marca como
// expansíveis (CallFunction, CallHTTP, ConsumeEvent). Para CallHTTP não
// resolvido, TargetResolved=false.
func buildExpansionSlot(n domain.Node, callTargetRaw any) *domain.ExpansionSlot {
	desc, ok := domain.DefaultRegistry.Get(n.Kind())
	if !ok || !desc.IsExpandable() {
		return nil
	}
	slot := &domain.ExpansionSlot{}

	// Preferimos o callTarget que já veio no record (evita segunda query).
	if callTarget, ok := asNeoNode(callTargetRaw); ok {
		if td, ok := nodeToDomainSafe(callTarget); ok {
			slot.TargetURN = td.URN()
			slot.TargetKind = td.Kind()
			slot.TargetResolved = td.Resolved()
			return slot
		}
	}

	// Sem callTarget no record: pode ser CallHTTP não linkado. Tenta ler
	// TargetURN da própria call; se vazio, marca como não resolvido.
	switch c := n.(type) {
	case *domain.CallFunction:
		slot.TargetURN = c.TargetURN
	case *domain.CallHTTP:
		if c.TargetURN != nil {
			slot.TargetURN = *c.TargetURN
		}
	case *domain.ConsumeEvent:
		slot.TargetURN = n.URN() // consume expande o próprio corpo
	}
	slot.TargetResolved = slot.TargetURN != ""
	return slot
}

// -------- ExpandFlow ---------------------------------------------------

func (r *Reader) ExpandFlow(ctx context.Context, targetURN domain.URN) (domain.FlowNode, error) {
	session := r.drv.NewSession(ctx, sessionConfig(r.database))
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypherExpandFlow, map[string]any{"targetURN": string(targetURN)})
		if err != nil {
			return nil, err
		}
		// Single retorna erro se não houver record — usamos Collect.
		return res.Collect(ctx)
	})
	if err != nil {
		return domain.FlowNode{}, fmt.Errorf("neo4j: ExpandFlow(%s): %w", targetURN, err)
	}

	recs := result.([]*neo4j.Record)
	if len(recs) == 0 {
		// Alvo não existe (ou não é expansível): stub inércia.
		return unresolvedFlowNode(targetURN), nil
	}
	rec := recs[0]

	targetNeo, ok := asNeoNode(recordGet(rec, "target"))
	if !ok {
		return unresolvedFlowNode(targetURN), nil
	}
	targetDomain, err := nodeToDomain(targetNeo)
	if err != nil {
		return domain.FlowNode{}, fmt.Errorf("decode expand target %q: %w", targetURN, err)
	}

	fn := domain.FlowNode{Node: targetDomain}
	children := asListProp(recordGet(rec, "children"))
	fn.Children = buildStepChildren(convertExpandChildren(children))
	return fn, nil
}

// convertExpandChildren adapta o formato de children do cypherExpandFlow
// (que não tem branches) para o shape ACHATADO que buildStepChildren espera.
func convertExpandChildren(children []any) []any {
	out := make([]any, 0, len(children))
	for _, c := range children {
		m := asMapProp(c)
		if m == nil {
			continue
		}
		out = append(out, map[string]any{
			"child":      m["child"],
			"idx":        m["idx"],
			"callTarget": m["callTarget"],
			// branch* ausentes = row sem branch, tratada como "só o child".
		})
	}
	return out
}

// unresolvedFlowNode constrói um FlowNode-stub para alvos que não existem
// no grafo (ex.: CallHTTP com destino externo ainda não extraído).
// Node.Resolved()=false sinaliza ao cliente que renderize como opaco.
func unresolvedFlowNode(urn domain.URN) domain.FlowNode {
	base := domain.BaseNode{
		URNValue:      urn,
		KindValue:     "Unknown",
		NameValue:     string(urn),
		ResolvedValue: false,
	}
	// Struct anônima que satisfaz Node via BaseNode embutido.
	return domain.FlowNode{
		Node: &stubNode{BaseNode: base},
	}
}

// stubNode é o Node placeholder devolvido quando ExpandFlow encontra um alvo
// não-resolvido. Existe só para satisfazer a interface Node.
type stubNode struct {
	domain.BaseNode
}

// -------- helpers de record --------------------------------------------

// recordGet lê um valor por chave do *neo4j.Record. Devolve nil se ausente.
func recordGet(rec *neo4j.Record, key string) any {
	v, ok := rec.Get(key)
	if !ok {
		return nil
	}
	return v
}

// ErrNotFound é devolvido pelo Reader quando o alvo (endpoint/serviço) não
// existe. Ainda não usado por todas as operações — reservado.
var ErrNotFound = errors.New("neo4j: node not found")
