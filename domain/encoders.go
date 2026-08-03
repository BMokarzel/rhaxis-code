package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// Este arquivo é o inverso de decoders.go: cada kind tem um EncodeFunc que
// converte a struct concreta em (labels, props) prontos para MERGE no Neo4j.
//
// A registration é feita no init() daqui, mutando os Descriptors já criados
// por decoders.go. Ordem dos inits: o Go garante que ambos rodem antes de
// qualquer código externo tocar o registry.

// --- helpers -------------------------------------------------------------

func encodeBase(b BaseNode) map[string]any {
	// resolved é escrito apenas pelos kinds que podem ser stubs (Call*/Consume);
	// para os demais o decoder default é true, então omitir mantém o contrato
	// sem inchar props em todos os nodes. Ver encodeCallFunction/CallHTTP/Consume.
	props := map[string]any{
		"urn":      string(b.URNValue),
		"kind":     string(b.KindValue),
		"name":     b.NameValue,
		"language": b.LanguageValue,
	}
	if b.ServiceURNValue != "" {
		props["serviceURN"] = string(b.ServiceURNValue)
	}
	if len(b.Metadata) > 0 {
		if raw, err := json.Marshal(b.Metadata); err == nil {
			props["metadata"] = string(raw)
		}
	}
	return props
}

// DisplayShapeFor mapeia um Kind numa forma canônica de render. Nomes ficam
// abstratos ("circle", "diamond") — o frontend traduz para o vocabulário
// da sua lib gráfica. Casos não cobertos devolvem "" (frontend usa default).
//
// Derivado no request time pela camada de view (ver cmd/rhaxis-api/view.go),
// NÃO persistido: seria duplicação nos milhares de nodes do grafo e
// dificultaria mudanças (rebuild do grafo inteiro só para trocar shape).
func DisplayShapeFor(k Kind) string {
	switch k {
	case KindEntry, KindExit, KindJoin:
		return "circle"
	case KindIfNode, KindSwitchNode:
		return "diamond"
	case KindTryNode:
		return "hexagon"
	case KindLoopNode:
		return "diamond"
	case KindReturn, KindThrow:
		return "triangle"
	case KindCallFunction, KindCallHTTP, KindCallDB:
		return "rect"
	case KindPublish, KindConsume:
		return "parallelogram"
	case KindLogNode:
		return "rect"
	case KindService:
		return "capsule"
	case KindDatabase, KindBroker:
		return "cylinder"
	case KindEndpoint:
		return "portal"
	case KindMethod, KindFunction, KindStruct, KindInterface, KindType, KindBlock:
		return "rect"
	case KindLanguage, KindFramework, KindRuntime:
		return "chip"
	case KindErrorType, KindMiddleware:
		return "shield"
	case KindConfig:
		return "note"
	}
	return ""
}

// DisplayCategoryFor bucketa o Kind para coloração/filtragem. Menos granular
// que Kind — vários kinds compartilham categoria (ex: CallHTTP e CallDB são
// ambos "io"). Serve para queries do tipo "todos os IO nodes deste endpoint".
//
// Derivado no request time pela camada de view (ver cmd/rhaxis-api/view.go).
func DisplayCategoryFor(k Kind) string {
	switch k {
	case KindEntry:
		return "start"
	case KindExit:
		return "end"
	case KindIfNode, KindSwitchNode, KindLoopNode, KindTryNode, KindJoin:
		return "control"
	case KindReturn, KindThrow:
		return "terminal"
	case KindCallFunction:
		return "call"
	case KindCallHTTP:
		return "io-http"
	case KindCallDB:
		return "io-db"
	case KindPublish, KindConsume:
		return "io-broker"
	case KindLogNode:
		return "log"
	case KindService:
		return "aggregator"
	case KindDatabase, KindBroker:
		return "infrastructure"
	case KindEndpoint:
		return "entrypoint"
	case KindMethod, KindFunction, KindBlock:
		return "container"
	case KindStruct, KindInterface, KindType:
		return "type"
	case KindLanguage, KindFramework, KindRuntime:
		return "tech"
	case KindErrorType:
		return "error"
	case KindConfig:
		return "config"
	case KindMiddleware:
		return "middleware"
	}
	return ""
}

// setIfNonEmpty facilita adicionar strings opcionais sem inflar o map.
func setIfNonEmpty(props map[string]any, key, value string) {
	if value != "" {
		props[key] = value
	}
}

// --- encoders ------------------------------------------------------------

func encodeService(n Node) ([]string, map[string]any, error) {
	s, ok := n.(*Service)
	if !ok {
		return nil, nil, fmt.Errorf("encodeService: expected *Service, got %T", n)
	}
	p := encodeBase(s.BaseNode)
	setIfNonEmpty(p, "repoURL", s.RepoURL)
	setIfNonEmpty(p, "framework", s.Framework)
	setIfNonEmpty(p, "runtime", s.Runtime)
	// Proveniência da última extração — só o Service carrega (todos os
	// demais nodes da extração compartilham este par, então replicar seria
	// duplicação).
	if !s.LastExtractedAt.IsZero() {
		p["lastExtractedAt"] = s.LastExtractedAt.Format(time.RFC3339)
	}
	setIfNonEmpty(p, "sourceRev", s.SourceRev)
	return []string{"Service"}, p, nil
}

func encodeDatabase(n Node) ([]string, map[string]any, error) {
	d := n.(*Database)
	p := encodeBase(d.BaseNode)
	setIfNonEmpty(p, "engine", d.Engine)
	return []string{"Database"}, p, nil
}

func encodeBroker(n Node) ([]string, map[string]any, error) {
	b := n.(*Broker)
	p := encodeBase(b.BaseNode)
	setIfNonEmpty(p, "engine", b.Engine)
	return []string{"Broker"}, p, nil
}

func encodeEndpoint(n Node) ([]string, map[string]any, error) {
	e := n.(*Endpoint)
	p := encodeBase(e.BaseNode)
	setIfNonEmpty(p, "httpMethod", e.HTTPMethod)
	setIfNonEmpty(p, "pathTemplate", e.PathTemplate)
	setIfNonEmpty(p, "framework", e.Framework)
	setIfNonEmpty(p, "handlerURN", string(e.HandlerURN))
	return []string{"Endpoint"}, p, nil
}

func encodeFunction(n Node) ([]string, map[string]any, error) {
	f := n.(*Function)
	p := encodeBase(f.BaseNode)
	setIfNonEmpty(p, "signature", f.Signature)
	p["isAsync"] = f.IsAsync
	return []string{"Function"}, p, nil
}

func encodeMethod(n Node) ([]string, map[string]any, error) {
	m := n.(*Method)
	p := encodeBase(m.BaseNode)
	setIfNonEmpty(p, "signature", m.Signature)
	p["isAsync"] = m.IsAsync
	setIfNonEmpty(p, "ownerTypeURN", string(m.OwnerTypeURN))
	return []string{"Method"}, p, nil
}

func encodeStruct(n Node) ([]string, map[string]any, error) {
	s := n.(*Struct)
	return []string{"Struct"}, encodeBase(s.BaseNode), nil
}

func encodeInterface(n Node) ([]string, map[string]any, error) {
	i := n.(*Interface)
	return []string{"Interface"}, encodeBase(i.BaseNode), nil
}

func encodeType(n Node) ([]string, map[string]any, error) {
	t := n.(*Type)
	p := encodeBase(t.BaseNode)
	setIfNonEmpty(p, "definition", t.Definition)
	return []string{"Type"}, p, nil
}

func encodeIfNode(n Node) ([]string, map[string]any, error) {
	i := n.(*IfNode)
	p := encodeBase(i.BaseNode)
	setIfNonEmpty(p, "conditionText", i.ConditionText)
	return []string{"IfNode"}, p, nil
}

func encodeSwitchNode(n Node) ([]string, map[string]any, error) {
	s := n.(*SwitchNode)
	p := encodeBase(s.BaseNode)
	setIfNonEmpty(p, "discriminant", s.Discriminant)
	return []string{"SwitchNode"}, p, nil
}

func encodeLoopNode(n Node) ([]string, map[string]any, error) {
	l := n.(*LoopNode)
	p := encodeBase(l.BaseNode)
	setIfNonEmpty(p, "loopKind", l.Kind_)
	return []string{"LoopNode"}, p, nil
}

func encodeTryNode(n Node) ([]string, map[string]any, error) {
	t := n.(*TryNode)
	return []string{"TryNode"}, encodeBase(t.BaseNode), nil
}

func encodeBlock(n Node) ([]string, map[string]any, error) {
	b := n.(*Block)
	return []string{"Block"}, encodeBase(b.BaseNode), nil
}

func encodeReturnNode(n Node) ([]string, map[string]any, error) {
	r := n.(*ReturnNode)
	p := encodeBase(r.BaseNode)
	setIfNonEmpty(p, "expressionText", r.ExpressionText)
	return []string{"ReturnNode"}, p, nil
}

func encodeThrowNode(n Node) ([]string, map[string]any, error) {
	t := n.(*ThrowNode)
	p := encodeBase(t.BaseNode)
	setIfNonEmpty(p, "expressionText", t.ExpressionText)
	return []string{"ThrowNode"}, p, nil
}

func encodeLanguage(n Node) ([]string, map[string]any, error) {
	l := n.(*Language)
	return []string{"Language"}, encodeBase(l.BaseNode), nil
}

func encodeFramework(n Node) ([]string, map[string]any, error) {
	f := n.(*Framework)
	return []string{"Framework"}, encodeBase(f.BaseNode), nil
}

func encodeRuntime(n Node) ([]string, map[string]any, error) {
	r := n.(*Runtime)
	return []string{"Runtime"}, encodeBase(r.BaseNode), nil
}

func encodeErrorType(n Node) ([]string, map[string]any, error) {
	e := n.(*ErrorType)
	p := encodeBase(e.BaseNode)
	setIfNonEmpty(p, "className", e.ClassName)
	if e.HTTPStatus != 0 {
		p["httpStatus"] = e.HTTPStatus
	}
	setIfNonEmpty(p, "origin", e.Origin)
	return []string{"ErrorType"}, p, nil
}

func encodeCallFunction(n Node) ([]string, map[string]any, error) {
	c := n.(*CallFunction)
	p := encodeBase(c.BaseNode)
	p["resolved"] = c.ResolvedValue
	setIfNonEmpty(p, "targetURN", string(c.TargetURN))
	return []string{"CallFunction"}, p, nil
}

func encodeCallHTTP(n Node) ([]string, map[string]any, error) {
	c := n.(*CallHTTP)
	p := encodeBase(c.BaseNode)
	p["resolved"] = c.ResolvedValue
	setIfNonEmpty(p, "httpMethod", c.HTTPMethod)
	setIfNonEmpty(p, "pathTemplate", c.PathTemplate)
	setIfNonEmpty(p, "targetHint", c.TargetHint)
	if c.TargetURN != nil {
		p["targetURN"] = string(*c.TargetURN)
	}
	return []string{"CallHTTP"}, p, nil
}

func encodeCallDB(n Node) ([]string, map[string]any, error) {
	c := n.(*CallDB)
	p := encodeBase(c.BaseNode)
	setIfNonEmpty(p, "operation", c.Operation)
	setIfNonEmpty(p, "targetURN", string(c.TargetURN))
	return []string{"CallDB"}, p, nil
}

func encodePublish(n Node) ([]string, map[string]any, error) {
	c := n.(*PublishEvent)
	p := encodeBase(c.BaseNode)
	setIfNonEmpty(p, "topic", c.Topic)
	setIfNonEmpty(p, "targetURN", string(c.TargetURN))
	return []string{"PublishEvent"}, p, nil
}

func encodeMiddleware(n Node) ([]string, map[string]any, error) {
	m := n.(*Middleware)
	p := encodeBase(m.BaseNode)
	setIfNonEmpty(p, "middlewareKind", m.Kind_)
	setIfNonEmpty(p, "phase", m.Phase)
	return []string{"Middleware"}, p, nil
}

func encodeConfig(n Node) ([]string, map[string]any, error) {
	c := n.(*Config)
	p := encodeBase(c.BaseNode)
	setIfNonEmpty(p, "key", c.Key)
	setIfNonEmpty(p, "category", c.Category)
	setIfNonEmpty(p, "defaultValue", c.DefaultValue)
	if c.Sensitive {
		p["sensitive"] = true
	}
	return []string{"Config"}, p, nil
}

func encodeLogNode(n Node) ([]string, map[string]any, error) {
	l := n.(*LogNode)
	p := encodeBase(l.BaseNode)
	setIfNonEmpty(p, "level", l.Level)
	setIfNonEmpty(p, "library", l.Library)
	setIfNonEmpty(p, "messageTemplate", l.MessageTemplate)
	if l.HasStructuredData {
		p["hasStructuredData"] = true
	}
	if l.IncludesTraceID {
		p["includesTraceId"] = true
	}
	return []string{"LogNode"}, p, nil
}

func encodeEntryNode(n Node) ([]string, map[string]any, error) {
	e := n.(*EntryNode)
	return []string{"EntryNode"}, encodeBase(e.BaseNode), nil
}

func encodeExitNode(n Node) ([]string, map[string]any, error) {
	e := n.(*ExitNode)
	return []string{"ExitNode"}, encodeBase(e.BaseNode), nil
}

func encodeJoinNode(n Node) ([]string, map[string]any, error) {
	j := n.(*JoinNode)
	p := encodeBase(j.BaseNode)
	setIfNonEmpty(p, "after", j.After)
	return []string{"JoinNode"}, p, nil
}

func encodeConsume(n Node) ([]string, map[string]any, error) {
	c := n.(*ConsumeEvent)
	p := encodeBase(c.BaseNode)
	p["resolved"] = c.ResolvedValue
	setIfNonEmpty(p, "topic", c.Topic)
	setIfNonEmpty(p, "targetURN", string(c.TargetURN))
	return []string{"ConsumeEvent"}, p, nil
}

// --- registration --------------------------------------------------------

// init roda depois do init de decoders.go (mesmo pacote, ordem alfabética
// dos arquivos: decoders.go antes de encoders.go). Aqui atualizamos cada
// Descriptor com seu Encoder.
func init() {
	pairs := map[Kind]EncodeFunc{
		KindService:      encodeService,
		KindDatabase:     encodeDatabase,
		KindBroker:       encodeBroker,
		KindEndpoint:     encodeEndpoint,
		KindFunction:     encodeFunction,
		KindMethod:       encodeMethod,
		KindStruct:       encodeStruct,
		KindInterface:    encodeInterface,
		KindType:         encodeType,
		KindIfNode:       encodeIfNode,
		KindSwitchNode:   encodeSwitchNode,
		KindLoopNode:     encodeLoopNode,
		KindTryNode:      encodeTryNode,
		KindBlock:        encodeBlock,
		KindReturn:       encodeReturnNode,
		KindThrow:        encodeThrowNode,
		KindLanguage:     encodeLanguage,
		KindFramework:    encodeFramework,
		KindRuntime:      encodeRuntime,
		KindErrorType:    encodeErrorType,
		KindLogNode:      encodeLogNode,
		KindConfig:       encodeConfig,
		KindMiddleware:   encodeMiddleware,
		KindEntry:        encodeEntryNode,
		KindExit:         encodeExitNode,
		KindJoin:         encodeJoinNode,
		KindCallFunction: encodeCallFunction,
		KindCallHTTP:     encodeCallHTTP,
		KindCallDB:       encodeCallDB,
		KindPublish:      encodePublish,
		KindConsume:      encodeConsume,
	}
	for kind, enc := range pairs {
		d, ok := DefaultRegistry.Get(kind)
		if !ok {
			// decoders.go não registrou este kind — bug de programação.
			panic(fmt.Sprintf("encoders: kind %q not previously registered", kind))
		}
		d.Encoder = enc
		DefaultRegistry.Register(d)
	}
}

// EncodeNode é o ponto de entrada para o Writer: dado um Node concreto,
// devolve labels adicionais e propriedades para persistir.
func EncodeNode(n Node) (labels []string, props map[string]any, err error) {
	d, ok := DefaultRegistry.Get(n.Kind())
	if !ok {
		return nil, nil, fmt.Errorf("encode: unknown kind %q", n.Kind())
	}
	if d.Encoder == nil {
		return nil, nil, fmt.Errorf("encode: no encoder for kind %q", n.Kind())
	}
	return d.Encoder(n)
}
