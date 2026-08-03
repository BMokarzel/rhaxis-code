package repository

import (
	"context"

	"github.com/BMokarzel/rhaxis-code.git/domain"
)

// Writer é o contrato de escrita no codegraph.
//
// Nota importante: o extrator externo (fora do escopo v1) é quem escreve o
// grosso do grafo. Este Writer serve para operações do CORE que precisam
// modificar o grafo pós-extração — hoje, na prática:
//   - o linker cross-service (resolver CallHTTP não-linkadas)
//   - materialização de :DEPENDS_ON
//   - correções pontuais
//
// Todas as operações são idempotentes (MERGE-based).
type Writer interface {
	// UpsertNode faz MERGE por urn. Labels e propriedades vêm do Encoder
	// registrado para o Kind do node. Se o node já existir, sobrescreve
	// todas as props (padrão SET =, não SET +=), garantindo que remover
	// campo em código resulta em remoção no banco.
	UpsertNode(ctx context.Context, n domain.Node) error

	// UpsertEdge faz MERGE de uma aresta dirigida entre dois URNs existentes.
	// Retorna erro se algum dos endpoints não existir (sem cascata implícita).
	// Props extras (ex.: {index: 3} em CONTAINS) sobrescrevem as anteriores.
	UpsertEdge(ctx context.Context, from, to domain.URN, kind domain.EdgeType, props map[string]any) error

	// DeleteNode remove um node e todas as arestas anexas. Idempotente
	// (não erra se o node não existir).
	DeleteNode(ctx context.Context, urn domain.URN) error

	// ResolveCallHTTPTarget é a operação atômica do linker cross-service.
	// Recebe uma CallHTTP ainda não linkada e o URN do Endpoint alvo real.
	// Executa em uma única transação:
	//   1. seta call.targetURN = target, call.resolved = true
	//   2. cria :EXPANDS_TO de call → target (affordance UI; v1 não emite
	//      :CALLS cross-service — invariante do design)
	//   3. garante :DEPENDS_ON {via:'http'} entre os dois serviços donos,
	//      incrementando weight se já existir. Skippa self-loop (sFrom = sTo).
	// Erra se call ou target não existirem, ou se não forem CallHTTP/Endpoint.
	ResolveCallHTTPTarget(ctx context.Context, callURN, targetEndpointURN domain.URN) error
}
