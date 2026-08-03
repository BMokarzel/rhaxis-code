// Package repository define os contratos de leitura do codegraph.
//
// Apenas contratos. Implementações vivem em infra/neo4j (ou outras infra/*).
// O core e o web dependem desta interface, nunca da implementação.
package repository

import (
	"context"

	"github.com/BMokarzel/rhaxis-code.git/domain"
)

// Reader agrupa as leituras que atendem às três telas do web e ao contrato
// de expansão lazy. Escritas ficam em outra interface (fora do escopo v1).
type Reader interface {
	// LoadServiceMap atende a Tela 1 (macro).
	// Retorna todos os serviços, sistemas externos (DBs, brokers) e as
	// dependências agregadas :DEPENDS_ON. NÃO retorna nodes de granularidade
	// fina; é uma visão agregada.
	//
	// filter=nil devolve tudo. Filtros ficam para uso futuro (busca).
	LoadServiceMap(ctx context.Context, filter *ServiceMapFilter) (domain.ServiceMap, error)

	// ListEndpoints atende a Tela 2 (serviço).
	// Dado o URN de um Service, lista seus endpoints (sem o corpo do fluxo).
	// Erro se o serviço não existir.
	ListEndpoints(ctx context.Context, serviceURN domain.URN) (domain.EndpointList, error)

	// LoadEndpointFlow atende a Tela 3 (endpoint).
	// Retorna a árvore inicial de fluxo do endpoint: um nível de :CONTAINS/
	// :BRANCH direto. Calls (function/http) vêm como FlowNode com
	// ExpansionSlot preenchido, MAS SEM o corpo do alvo. O corpo é obtido
	// via ExpandFlow.
	//
	// Erro se o endpoint não existir.
	LoadEndpointFlow(ctx context.Context, endpointURN domain.URN) (domain.EndpointFlow, error)

	// ExpandFlow é o contrato de expansão lazy.
	// Dado o URN alvo de um ExpansionSlot, retorna a próxima "camada": o
	// fluxo interno do alvo (Function/Method/Endpoint), com seus próprios
	// ExpansionSlots preenchidos para o clique seguinte.
	//
	// Profundidade fixa = 1. Nunca segue :CALLS transitivamente — recursão
	// é decisão do cliente.
	//
	// Se o alvo não existir OU não estiver resolvido (stub HTTP), retorna
	// um FlowNode com Node.Resolved()=false e sem children. O cliente decide
	// como renderizar.
	ExpandFlow(ctx context.Context, targetURN domain.URN) (domain.FlowNode, error)
}

// ServiceMapFilter é opcional em LoadServiceMap.
// Filtros vazios significam "sem filtro naquela dimensão".
type ServiceMapFilter struct {
	NamePrefix string
	Languages  []string
}
