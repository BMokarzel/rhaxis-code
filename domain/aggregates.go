package domain

// Este arquivo define os agregados conceituais do domain — estruturas que NÃO
// são nodes 1:1 no grafo, mas sim views montadas pelo repository para atender
// diretamente às três telas do web e ao contrato de expansão lazy.

// --- Tela 1 --------------------------------------------------------------

// ServiceMap é o resultado da Tela 1 (macro).
// Nenhum node de granularidade fina aparece aqui — só serviços, sistemas
// externos e as dependências agregadas (:DEPENDS_ON materializada).
type ServiceMap struct {
	Services        []Service
	ExternalSystems []Node // Databases, Brokers (heterogêneo por design)
	Dependencies    []ServiceDependency
}

// ServiceDependency é uma aresta agregada entre dois serviços.
// Materializada pelo linker cross-service; a query da Tela 1 apenas lê.
type ServiceDependency struct {
	From   URN
	To     URN
	Via    string // "http" | "event" | "db"
	Weight int    // nº de calls concretas por trás; 0 quando não computado
}

// --- Tela 2 --------------------------------------------------------------

// EndpointList é o resultado da Tela 2 (serviço).
// Traz os endpoints sem qualquer corpo de fluxo — só metadados de rota.
type EndpointList struct {
	Service   Service
	Endpoints []Endpoint
}

// --- Tela 3 e expansão lazy ---------------------------------------------

// FlowNode é uma célula do diagrama da Tela 3.
// NÃO é um Node do grafo — é uma view que carrega o Node + a estrutura de
// exibição (children ordenados, branches nomeados, slot de expansão).
type FlowNode struct {
	Node     Node
	Children []FlowNode // via :CONTAINS ordenado por index; vazio em folhas

	// Branches é preenchido apenas para IfNode/SwitchNode/TryNode.
	// Chave: label da branch ("then", "else", "case:foo", "default", "catch", ...).
	Branches map[string][]FlowNode

	// Expansion, quando não-nil, indica que este node aponta para um subgrafo
	// que ainda não foi carregado (CallFunction, CallHTTP, ConsumeEvent, ...).
	Expansion *ExpansionSlot
}

// ExpansionSlot é o "clique aqui para abrir" — contrato de expansão lazy.
// O cliente decide se e quando chamar repository.Reader.ExpandFlow(TargetURN).
type ExpansionSlot struct {
	TargetURN      URN
	TargetKind     Kind
	TargetResolved bool // false = stub (CallHTTP sem link); render diferente
}

// EndpointFlow é a árvore inicial servida à Tela 3.
// Root é um FlowNode sintético que agrega os steps diretos do Endpoint.
type EndpointFlow struct {
	Endpoint Endpoint
	Root     FlowNode
}
