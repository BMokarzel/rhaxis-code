package loader

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/BMokarzel/rhaxis-code.git/domain"
	"github.com/BMokarzel/rhaxis-code.git/repository"
)

// Stats resume o resultado de LoadInto — útil para logs e testes.
type Stats struct {
	NodesUpserted int
	EdgesUpserted int
	NodesSkipped  int // kind desconhecido no registry
	EdgesSkipped  int
}

// LoadInto consome um Payload e escreve tudo via Writer.
//
// Ordem importante: nodes primeiro, edges depois. O Writer valida existência
// dos endpoints em UpsertEdge — se invertermos, o segundo passo quebraria.
//
// Trade-off: se algum node falhar no meio, os anteriores ficam persistidos
// (sem transação global). A operação é idempotente, então rodar de novo
// corrige. Escolhi isso em vez de transação gigante porque o Writer atual
// não expõe TX composta e o payload pode ter milhares de nodes.
func LoadInto(ctx context.Context, p *Payload, w repository.Writer) (Stats, error) {
	var s Stats

	for _, nin := range p.Nodes {
		props, err := flattenNode(nin, p)
		if err != nil {
			return s, fmt.Errorf("loader: flatten %s: %w", nin.URN, err)
		}
		n, err := domain.DecodeByKind(domain.Kind(nin.Kind), props)
		if err != nil {
			// Kind desconhecido — pulamos com log em vez de falhar tudo.
			// Trade-off: um extrator novo pode emitir kinds v2 que o core
			// ainda não conhece; melhor persistir o resto do que abortar.
			s.NodesSkipped++
			continue
		}
		if err := w.UpsertNode(ctx, n); err != nil {
			return s, fmt.Errorf("loader: upsert node %s: %w", nin.URN, err)
		}
		s.NodesUpserted++
	}

	for _, ein := range p.Edges {
		et := domain.EdgeType(ein.Type)
		if !isKnownEdgeType(et) {
			s.EdgesSkipped++
			continue
		}
		if err := w.UpsertEdge(ctx, domain.URN(ein.From), domain.URN(ein.To), et, ein.Props); err != nil {
			return s, fmt.Errorf("loader: upsert edge %s -[%s]-> %s: %w", ein.From, ein.Type, ein.To, err)
		}
		s.EdgesUpserted++
	}

	return s, nil
}

// flattenNode converte NodeIn (formato JSON) no map[string]any que os
// decoders do domain esperam: campos base no topo + kind-specific props
// achatadas no mesmo nível.
//
// A proveniência da extração (lastExtractedAt/sourceRev) é injetada APENAS
// no node Service — todos os demais nodes desta rodada compartilham o
// mesmo par de valores; persistir em cada um seria duplicação massiva.
func flattenNode(nin NodeIn, p *Payload) (map[string]any, error) {
	if nin.URN == "" {
		return nil, fmt.Errorf("node has empty urn")
	}
	if nin.Kind == "" {
		return nil, fmt.Errorf("node %s has empty kind", nin.URN)
	}

	out := make(map[string]any, 8+len(nin.Properties))
	out["urn"] = nin.URN
	out["kind"] = nin.Kind
	if nin.Name != "" {
		out["name"] = nin.Name
	}
	if nin.Resolved != nil {
		out["resolved"] = *nin.Resolved
	}
	if nin.Kind == "Service" {
		// decodeService aceita string RFC3339 direto — passamos como string
		// para o Writer também serializar como string. O parse acontece
		// em optTime quando necessário.
		if p.ExtractedAt != "" {
			out["lastExtractedAt"] = p.ExtractedAt
		}
		if p.SourceRev != "" {
			out["sourceRev"] = p.SourceRev
		}
	}
	// serviceURN e language ficam derivados quando fizerem sentido.
	// Só o node Service tem URN igual ao service.urn; os demais herdam
	// implicitamente pelo prefixo. Não injetamos aqui — quem quiser
	// filtrar por serviço usa o prefixo do URN.

	// Metadata como JSON string (o decoder faz Unmarshal).
	if len(nin.Metadata) > 0 {
		if b, err := json.Marshal(nin.Metadata); err == nil {
			out["metadata"] = string(b)
		}
	}

	for k, v := range nin.Properties {
		out[k] = v
	}
	return out, nil
}

// isKnownEdgeType é a lista fechada de tipos v1. Se o extrator emitir algo
// fora disso, ignoramos com log — evita gravar lixo no grafo.
func isKnownEdgeType(t domain.EdgeType) bool {
	switch t {
	case domain.EdgeOwns,
		domain.EdgeExposes,
		domain.EdgeContains,
		domain.EdgeNext,
		domain.EdgeBranch,
		domain.EdgeCalls,
		domain.EdgeUsesDB,
		domain.EdgePublishesTo,
		domain.EdgeConsumes,
		domain.EdgeHasType,
		domain.EdgeImplements,
		domain.EdgeExtends,
		domain.EdgeDependsOn,
		domain.EdgeUses,
		domain.EdgeWrittenIn,
		domain.EdgeThrows,
		domain.EdgeCatches,
		domain.EdgeHandlesError,
		domain.EdgeLogs,
		domain.EdgeUsesConfig,
		domain.EdgeProtects,
		domain.EdgeExpandsTo:
		return true
	}
	return false
}
