// Package domain contém o modelo conceitual do codegraph.
//
// O domain é agnóstico de persistência: não importa neo4j, cypher ou driver.
// Toda tradução node <-> struct vive na camada infra/neo4j, mediada pelo
// Registry deste pacote.
package domain

// URN é a identidade estável e globalmente única de um node.
// É opaca ao consumidor: só se compara por igualdade e se resolve por repositório.
// Formato canônico: urn:cg:<service>:<language>:<kind>:<qualified-path>[@<disambiguator>].
type URN string

// Kind identifica o tipo do node de forma agnóstica à label do Neo4j.
type Kind string

// Kinds v1. Adicionar um novo kind é registrar um Descriptor no Registry —
// nenhuma query ou mapper existente precisa mudar.
const (
	KindService      Kind = "Service"
	KindDatabase     Kind = "Database"
	KindBroker       Kind = "Broker"
	KindEndpoint     Kind = "Endpoint"
	KindFunction     Kind = "Function"
	KindMethod       Kind = "Method"
	KindStruct       Kind = "Struct"
	KindInterface    Kind = "Interface"
	KindType         Kind = "Type"
	KindIfNode       Kind = "IfNode"
	KindSwitchNode   Kind = "SwitchNode"
	KindLoopNode     Kind = "LoopNode"
	KindTryNode      Kind = "TryNode"
	KindBlock        Kind = "Block"
	KindCallFunction Kind = "CallFunction"
	KindCallHTTP     Kind = "CallHTTP"
	KindCallDB       Kind = "CallDB"
	KindPublish      Kind = "PublishEvent"
	KindConsume      Kind = "ConsumeEvent"
	KindReturn       Kind = "ReturnNode"
	KindThrow        Kind = "ThrowNode"
	KindLanguage     Kind = "Language"
	KindFramework    Kind = "Framework"
	KindRuntime      Kind = "Runtime"
	// KindErrorType cataloga uma classe de erro (HttpException, NotFoundException,
	// etc). Alvo de THROWS/CATCHES/HANDLES_ERROR. URN é global (`urn:cg:_tech:error:X`)
	// para que serviços diferentes lançando a mesma classe compartilhem o node —
	// habilita queries topológicas "quem lança NotFoundException?".
	KindErrorType Kind = "ErrorType"
	// KindLogNode é um step de log emitido dentro do fluxo (console.log, Logger.info,
	// winston/pino, etc). Fica como filho do container executável (Endpoint/Method)
	// via CONTAINS + NEXT — igual aos outros steps — e ainda expõe uma edge LOGS
	// vinda do Function/Method dono, pra facilitar queries "quem loga na
	// função X?" sem traversal por CONTAINS.
	KindLogNode Kind = "LogNode"
	// KindConfig cataloga uma chave de configuração/env var lida pelo serviço
	// (process.env.X, configService.get('X')). URN global (`urn:cg:_tech:config:X`)
	// para compartilhar node entre serviços que leem a mesma chave —
	// habilita "quem lê DATABASE_URL?".
	KindConfig Kind = "Config"
	// KindMiddleware representa um guard/interceptor/pipe/filter (NestJS) ou
	// equivalente em outros frameworks. Applied a Endpoints via PROTECTS.
	// Per-serviço (classes com mesmo nome em serviços diferentes são
	// implementações distintas).
	KindMiddleware Kind = "Middleware"
	// KindEntry/KindExit marcam início/fim do fluxo de execução de um
	// container (Endpoint/Function/Method). Emitidos como primeiro e último
	// CONTAINS child do body Block. Servem de âncora visual para o front
	// (pin de start/end no render de fluxo) e ponto natural para futuras
	// props derivadas (tempo médio, taxa de sucesso, etc). URN =
	// <containerURN>.entry / <containerURN>.exit.
	KindEntry Kind = "EntryNode"
	KindExit  Kind = "ExitNode"
	// KindJoin marca o ponto de re-convergência de controle depois de
	// IfNode/SwitchNode/TryNode. Emitido apenas quando há statements
	// depois do branchy no mesmo escopo — Join sozinho no fim seria
	// redundante com Exit. Sibling dos steps do body, ligado por NEXT
	// entre o branchy e o próximo statement.
	KindJoin Kind = "JoinNode"
)

// Node é o contrato mínimo. Todo node concreto embute BaseNode.
type Node interface {
	URN() URN
	Kind() Kind
	Name() string
	ServiceURN() URN
	Language() string
	Resolved() bool
}

// BaseNode carrega o contrato comum. Concretos embutem por composição
// e ganham os métodos do Node interface automaticamente.
//
// Proveniência (extractedAt/sourceRev) foi movida para o Service (campo
// LastExtractedAt/SourceRev). Todos os nodes de uma extração compartilham
// os mesmos valores, então persistir em N nodes era duplicação pura.
type BaseNode struct {
	URNValue        URN
	KindValue       Kind
	NameValue       string
	ServiceURNValue URN
	LanguageValue   string
	ResolvedValue   bool
	// Metadata guarda props não-tipadas do kind (extensão sem quebrar o schema Go).
	// Persistido como string JSON no Neo4j; decodificado para map[string]any aqui.
	Metadata map[string]any
}

func (b BaseNode) URN() URN         { return b.URNValue }
func (b BaseNode) Kind() Kind       { return b.KindValue }
func (b BaseNode) Name() string     { return b.NameValue }
func (b BaseNode) ServiceURN() URN  { return b.ServiceURNValue }
func (b BaseNode) Language() string { return b.LanguageValue }
func (b BaseNode) Resolved() bool   { return b.ResolvedValue }

// EdgeType enumera os tipos de aresta persistidos no grafo.
// Usado pelo Registry para descrever conexões válidas e pelo mapper para
// filtrar/desambiguar edges nas queries.
type EdgeType string

const (
	EdgeOwns        EdgeType = "OWNS"
	EdgeExposes     EdgeType = "EXPOSES"
	EdgeContains    EdgeType = "CONTAINS"
	EdgeNext        EdgeType = "NEXT"
	EdgeBranch      EdgeType = "BRANCH"
	EdgeCalls       EdgeType = "CALLS"
	EdgeUsesDB      EdgeType = "USES_DB"
	EdgePublishesTo EdgeType = "PUBLISHES_TO"
	EdgeConsumes    EdgeType = "CONSUMES_FROM"
	EdgeHasType     EdgeType = "HAS_TYPE"
	EdgeImplements  EdgeType = "IMPLEMENTS"
	EdgeExtends     EdgeType = "EXTENDS"
	EdgeDependsOn   EdgeType = "DEPENDS_ON"
	// Tech edges: Service -> Language/Framework/Runtime nodes globais.
	EdgeUses      EdgeType = "USES"
	EdgeWrittenIn EdgeType = "WRITTEN_IN"
	// Error edges: catálogo de erros do fluxo.
	//   THROWS         Function/ThrowNode -> ErrorType   (com prop caughtInternally)
	//   CATCHES        TryNode            -> ErrorType   (com prop via = catch label)
	//   HANDLES_ERROR  Endpoint           -> ErrorType   (com prop httpStatus opcional)
	EdgeThrows       EdgeType = "THROWS"
	EdgeCatches      EdgeType = "CATCHES"
	EdgeHandlesError EdgeType = "HANDLES_ERROR"
	// EdgeLogs: Function/Method -> LogNode. Redundante com CONTAINS+NEXT
	// pra propósito de query direta.
	EdgeLogs EdgeType = "LOGS"
	// EdgeUsesConfig: Function/Method -> Config. Dependência declarativa
	// pra queries "quais funções leem esta env var?".
	EdgeUsesConfig EdgeType = "USES_CONFIG"
	// EdgeProtects: Middleware -> Endpoint. Guard/interceptor/pipe/filter
	// aplicado ao endpoint (via decorator @UseGuards etc, na classe ou método).
	EdgeProtects EdgeType = "PROTECTS"
	// EdgeExpandsTo: Call* -> target (Method/Function/Endpoint). Affordance
	// UI ortogonal a CALLS — enquanto CALLS é a relação semântica de fluxo
	// de controle, EXPANDS_TO responde "o que revelar quando o usuário
	// expandir esse call node no render". Coincidem para CallFunction
	// resolvido; para CallHTTP resolvido pelo linker cross-service, só
	// EXPANDS_TO existe (não há CALLS cross-service em v1).
	EdgeExpandsTo EdgeType = "EXPANDS_TO"
)
