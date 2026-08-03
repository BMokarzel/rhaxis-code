package domain

import "time"

// Este arquivo agrupa as structs concretas dos kinds. Cada uma embute BaseNode
// e adiciona apenas as propriedades específicas que o domain precisa expor.
// Props raras ou não-consultadas devem ficar em BaseNode.Metadata.

// --- Aggregators ---------------------------------------------------------

// Service é um microserviço. Raiz de agrupamento no macro-view (Tela 1).
//
// LastExtractedAt e SourceRev são atributos da última extração deste serviço.
// Ficam aqui (e não no BaseNode) porque todos os nodes de uma extração
// compartilham o mesmo par — replicar em cada node seria duplicação massiva.
type Service struct {
	BaseNode
	RepoURL         string
	Framework       string // "nestjs", "gin", ...
	Runtime         string // "node18", "go1.22", ...
	LastExtractedAt time.Time
	SourceRev       string
}

// Database representa um banco externo consumido por um ou mais serviços.
type Database struct {
	BaseNode
	Engine string // "postgres", "mongo", ...
}

// Broker representa um message broker (Kafka, RabbitMQ, etc).
type Broker struct {
	BaseNode
	Engine string
}

// --- Code containers -----------------------------------------------------

// Endpoint é um ponto de entrada HTTP. Raiz do fluxo da Tela 3.
type Endpoint struct {
	BaseNode
	HTTPMethod   string
	PathTemplate string
	Framework    string
	HandlerURN   URN // ponteiro para a Function/Method que efetivamente serve
}

// Function representa uma unidade de execução nomeada.
type Function struct {
	BaseNode
	Signature string
	IsAsync   bool
}

// Method é uma função ligada a um tipo (struct/class).
// Embute Function para reuso de decoder, ciente do trade-off documentado
// em §7.2 do doc de design (polimorfismo pobre em Go).
type Method struct {
	Function
	OwnerTypeURN URN
}

// Struct representa uma struct/class extraída.
type Struct struct {
	BaseNode
	Fields  []FieldSlot
	Methods []URN // URNs dos Methods deste tipo
}

// Interface representa um contrato de tipo.
type Interface struct {
	BaseNode
	Methods []URN
}

// Type representa alias/union/enum e afins.
type Type struct {
	BaseNode
	Definition string
}

// FieldSlot é um campo de Struct.
type FieldSlot struct {
	Name    string
	TypeURN URN
}

// --- Control flow --------------------------------------------------------

// IfNode. Ramificações via :BRANCH para Blocks (label "then"/"else").
type IfNode struct {
	BaseNode
	ConditionText string // <= ~200 chars, para display
}

// SwitchNode. Ramificações via :BRANCH com label "case:<x>" ou "default".
type SwitchNode struct {
	BaseNode
	Discriminant string
}

// LoopNode. Corpo do loop é o único :BRANCH com label "body".
type LoopNode struct {
	BaseNode
	Kind_ string // "for", "while", "for-of", ... (evita colisão com Kind())
}

// TryNode. Ramificações com labels "try", "catch:<T>", "finally".
type TryNode struct {
	BaseNode
}

// Block é um contêiner ordenado de steps de execução.
// Aparece como filho direto de Endpoint/Function/Method ou como alvo de :BRANCH.
type Block struct {
	BaseNode
}

// ReturnNode representa um `return <expr>` do fluxo — nó terminal de uma
// sequência (nenhum :NEXT sai dele). ExpressionText guarda o texto original
// para display; se a expressão contiver uma call materializável, essa call
// é emitida como step irmão imediatamente antes.
type ReturnNode struct {
	BaseNode
	ExpressionText string
}

// ThrowNode é análogo ao ReturnNode mas para `throw`. Também terminal.
type ThrowNode struct {
	BaseNode
	ExpressionText string
}

// EntryNode / ExitNode delimitam o fluxo de um container executável.
// Sem campos próprios — a identidade (URN + Kind) já basta. Ficam como
// primeiro e último CONTAINS child do body Block do container, ligados por
// NEXT ao primeiro/último step "real". BaseNode é suficiente.
type EntryNode struct {
	BaseNode
}

type ExitNode struct {
	BaseNode
}

// JoinNode marca a re-convergência de controle após um IfNode/TryNode/
// SwitchNode. `After` guarda o tipo do branchy que o precede ("if",
// "try", "switch") para facilitar display sem traversal reverso.
type JoinNode struct {
	BaseNode
	After string
}

// --- Tech (globais, sem serviceURN) --------------------------------------

// Language, Framework e Runtime são nodes globais compartilhados por N
// serviços. URN sem slug (`urn:cg:_tech:...`) garante MERGE idempotente
// cross-serviço. Usados por queries topológicas ("todos os serviços que
// falam nestjs") sem depender de propriedade escalar no Service.
type Language struct {
	BaseNode
}

type Framework struct {
	BaseNode
}

type Runtime struct {
	BaseNode
}

// ErrorType representa uma classe de erro lançada/capturada no código
// (HttpException, NotFoundException, TypeError...). Global (sem serviceURN),
// para que múltiplos serviços que usam a mesma classe compartilhem o node.
//
// HTTPStatus é preenchido quando o extrator consegue mapear (ex: HttpException
// do NestJS carrega status no construtor). Origin classifica origem para
// filtragem em queries: "builtin" (Error, TypeError...), "framework" (NestJS,
// Express), "user" (definida no próprio codebase).
type ErrorType struct {
	BaseNode
	ClassName  string
	HTTPStatus int
	Origin     string
}

// Middleware representa um guard/interceptor/pipe/filter aplicado a
// endpoints via decorator (@UseGuards, @UseInterceptors, ...). Kind_
// distingue subtipo, Phase indica se roda pre ou post handler.
type Middleware struct {
	BaseNode
	Kind_ string // "guard" | "interceptor" | "pipe" | "filter"
	Phase string // "pre" | "post"
}

// Config representa uma chave de configuração lida pelo serviço:
// `process.env.X`, `configService.get('X')`. Global (sem serviceURN) — dois
// serviços que leem a mesma chave compartilham o node.
// Category: "env" | "config" | "secret". DefaultValue guarda o fallback
// textual quando detectado (`process.env.X ?? 'default'`). Sensitive é
// heurística por nome (contém SECRET, TOKEN, KEY, PASS).
type Config struct {
	BaseNode
	Key          string
	Category     string
	DefaultValue string
	Sensitive    bool
}

// LogNode é um step de log dentro do fluxo (console.log, Logger.info, winston/pino).
// Level é opcional (info/warn/error/debug); Library identifica a biblioteca
// (`console`, `winston`, `pino`, `nest-logger`) para queries "quem usa X para
// logar?". MessageTemplate guarda a primeira string literal do argumento
// (best-effort) — útil para achar padrões de log sensíveis sem inferência
// dinâmica.
type LogNode struct {
	BaseNode
	Level              string
	Library            string
	MessageTemplate    string
	HasStructuredData  bool
	IncludesTraceID    bool
}

// --- Calls ---------------------------------------------------------------

// CallFunction aponta para uma Function/Method interna ao grafo (mesmo serviço
// ou outro que já foi extraído).
type CallFunction struct {
	BaseNode
	TargetURN URN
}

// CallHTTP: alvo pode não existir ainda. TargetURN nulo => call para fora do
// grafo ou pendente de resolução pelo linker cross-service.
type CallHTTP struct {
	BaseNode
	HTTPMethod   string
	PathTemplate string
	TargetURN    *URN   // resolvido pelo linker, nil se ainda não linkado
	TargetHint   string // pista textual (host header, config key)
}

// CallDB representa uma operação contra um Database.
type CallDB struct {
	BaseNode
	Operation string // "select" | "insert" | "update" | "delete" | "raw"
	TargetURN URN    // Database
}

// PublishEvent: emite um evento para um Broker/topic.
type PublishEvent struct {
	BaseNode
	Topic     string
	TargetURN URN // Broker
}

// ConsumeEvent: consome um evento de um Broker/topic. Costuma ser raiz de um
// fluxo similar a Endpoint (handler assíncrono).
type ConsumeEvent struct {
	BaseNode
	Topic     string
	TargetURN URN // Broker
}
