package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// Este arquivo registra os Descriptors de todos os kinds v1 no DefaultRegistry.
// Cada Descriptor define seu Decoder — a função que converte um map de
// propriedades vindo do driver Neo4j numa struct concreta do domain.
//
// Regras dos decoders:
//   - Populam BaseNode via decodeBase(props) e as próprias props em seguida.
//   - Props desconhecidas ficam em BaseNode.Metadata.
//   - Erro em decodeBase é fatal; erro em prop opcional é ignorado
//     (o node ainda é útil mesmo sem esses campos).

// --- helpers -------------------------------------------------------------

func decodeBase(props map[string]any) (BaseNode, error) {
	urn, err := requireString(props, "urn")
	if err != nil {
		return BaseNode{}, err
	}
	kind, err := requireString(props, "kind")
	if err != nil {
		return BaseNode{}, err
	}
	base := BaseNode{
		URNValue:        URN(urn),
		KindValue:       Kind(kind),
		NameValue:       optString(props, "name"),
		ServiceURNValue: URN(optString(props, "serviceURN")),
		LanguageValue:   optString(props, "language"),
		ResolvedValue:   optBool(props, "resolved", true),
		Metadata:        optJSONMap(props, "metadata"),
	}
	return base, nil
}

func requireString(props map[string]any, key string) (string, error) {
	v, ok := props[key]
	if !ok {
		return "", fmt.Errorf("missing required property %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("property %q is not a string (got %T)", key, v)
	}
	return s, nil
}

func optString(props map[string]any, key string) string {
	if v, ok := props[key].(string); ok {
		return v
	}
	return ""
}

func optBool(props map[string]any, key string, def bool) bool {
	if v, ok := props[key].(bool); ok {
		return v
	}
	return def
}

// optTime aceita time.Time (driver v5 já converte) ou string RFC3339.
func optTime(props map[string]any, key string) time.Time {
	switch v := props[key].(type) {
	case time.Time:
		return v
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
	}
	return time.Time{}
}

// optInt aceita int, int64 e float64 (driver Neo4j pode devolver qualquer
// dos três dependendo da versão/serialização). Zero se ausente ou não-numérico.
func optInt(props map[string]any, key string) int {
	switch v := props[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func optJSONMap(props map[string]any, key string) map[string]any {
	s, ok := props[key].(string)
	if !ok || s == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil // metadata corrompida não deve derrubar o node
	}
	return out
}

func optURNPtr(props map[string]any, key string) *URN {
	s := optString(props, key)
	if s == "" {
		return nil
	}
	u := URN(s)
	return &u
}

// --- decoders ------------------------------------------------------------

func decodeService(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Service{
		BaseNode:        base,
		RepoURL:         optString(props, "repoURL"),
		Framework:       optString(props, "framework"),
		Runtime:         optString(props, "runtime"),
		LastExtractedAt: optTime(props, "lastExtractedAt"),
		SourceRev:       optString(props, "sourceRev"),
	}, nil
}

func decodeDatabase(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Database{BaseNode: base, Engine: optString(props, "engine")}, nil
}

func decodeBroker(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Broker{BaseNode: base, Engine: optString(props, "engine")}, nil
}

func decodeEndpoint(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Endpoint{
		BaseNode:     base,
		HTTPMethod:   optString(props, "httpMethod"),
		PathTemplate: optString(props, "pathTemplate"),
		Framework:    optString(props, "framework"),
		HandlerURN:   URN(optString(props, "handlerURN")),
	}, nil
}

func decodeFunction(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Function{
		BaseNode:  base,
		Signature: optString(props, "signature"),
		IsAsync:   optBool(props, "isAsync", false),
	}, nil
}

func decodeMethod(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Method{
		Function: Function{
			BaseNode:  base,
			Signature: optString(props, "signature"),
			IsAsync:   optBool(props, "isAsync", false),
		},
		OwnerTypeURN: URN(optString(props, "ownerTypeURN")),
	}, nil
}

func decodeStruct(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	// Fields e Methods não são recuperados aqui — são edges (:HAS_TYPE, :OWNS).
	// O reader popula quando necessário; por default ficam vazios.
	return &Struct{BaseNode: base}, nil
}

func decodeInterface(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Interface{BaseNode: base}, nil
}

func decodeType(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Type{BaseNode: base, Definition: optString(props, "definition")}, nil
}

func decodeIfNode(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &IfNode{BaseNode: base, ConditionText: optString(props, "conditionText")}, nil
}

func decodeSwitchNode(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &SwitchNode{BaseNode: base, Discriminant: optString(props, "discriminant")}, nil
}

func decodeLoopNode(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &LoopNode{BaseNode: base, Kind_: optString(props, "loopKind")}, nil
}

func decodeTryNode(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &TryNode{BaseNode: base}, nil
}

func decodeBlock(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Block{BaseNode: base}, nil
}

func decodeReturnNode(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &ReturnNode{BaseNode: base, ExpressionText: optString(props, "expressionText")}, nil
}

func decodeThrowNode(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &ThrowNode{BaseNode: base, ExpressionText: optString(props, "expressionText")}, nil
}

func decodeLanguage(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Language{BaseNode: base}, nil
}

func decodeFramework(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Framework{BaseNode: base}, nil
}

func decodeRuntime(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Runtime{BaseNode: base}, nil
}

func decodeCallFunction(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &CallFunction{BaseNode: base, TargetURN: URN(optString(props, "targetURN"))}, nil
}

func decodeCallHTTP(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &CallHTTP{
		BaseNode:     base,
		HTTPMethod:   optString(props, "httpMethod"),
		PathTemplate: optString(props, "pathTemplate"),
		TargetURN:    optURNPtr(props, "targetURN"),
		TargetHint:   optString(props, "targetHint"),
	}, nil
}

func decodeCallDB(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &CallDB{
		BaseNode:  base,
		Operation: optString(props, "operation"),
		TargetURN: URN(optString(props, "targetURN")),
	}, nil
}

func decodePublish(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &PublishEvent{
		BaseNode:  base,
		Topic:     optString(props, "topic"),
		TargetURN: URN(optString(props, "targetURN")),
	}, nil
}

func decodeErrorType(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &ErrorType{
		BaseNode:   base,
		ClassName:  optString(props, "className"),
		HTTPStatus: optInt(props, "httpStatus"),
		Origin:     optString(props, "origin"),
	}, nil
}

func decodeMiddleware(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Middleware{
		BaseNode: base,
		Kind_:    optString(props, "middlewareKind"),
		Phase:    optString(props, "phase"),
	}, nil
}

func decodeConfig(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &Config{
		BaseNode:     base,
		Key:          optString(props, "key"),
		Category:     optString(props, "category"),
		DefaultValue: optString(props, "defaultValue"),
		Sensitive:    optBool(props, "sensitive", false),
	}, nil
}

func decodeLogNode(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &LogNode{
		BaseNode:          base,
		Level:             optString(props, "level"),
		Library:           optString(props, "library"),
		MessageTemplate:   optString(props, "messageTemplate"),
		HasStructuredData: optBool(props, "hasStructuredData", false),
		IncludesTraceID:   optBool(props, "includesTraceId", false),
	}, nil
}

func decodeEntryNode(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &EntryNode{BaseNode: base}, nil
}

func decodeExitNode(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &ExitNode{BaseNode: base}, nil
}

func decodeJoinNode(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &JoinNode{BaseNode: base, After: optString(props, "after")}, nil
}

func decodeConsume(props map[string]any) (Node, error) {
	base, err := decodeBase(props)
	if err != nil {
		return nil, err
	}
	return &ConsumeEvent{
		BaseNode:  base,
		Topic:     optString(props, "topic"),
		TargetURN: URN(optString(props, "targetURN")),
	}, nil
}

// --- registration --------------------------------------------------------

func init() {
	reg := DefaultRegistry
	// containers e agregadores
	reg.Register(Descriptor{Kind: KindService, Labels: []string{"Service"}, Decoder: decodeService})
	reg.Register(Descriptor{Kind: KindDatabase, Labels: []string{"Database"}, Decoder: decodeDatabase})
	reg.Register(Descriptor{Kind: KindBroker, Labels: []string{"Broker"}, Decoder: decodeBroker})
	reg.Register(Descriptor{Kind: KindEndpoint, Labels: []string{"Endpoint"}, Decoder: decodeEndpoint})
	reg.Register(Descriptor{Kind: KindFunction, Labels: []string{"Function"}, Decoder: decodeFunction})
	reg.Register(Descriptor{Kind: KindMethod, Labels: []string{"Method"}, Decoder: decodeMethod})
	reg.Register(Descriptor{Kind: KindStruct, Labels: []string{"Struct"}, Decoder: decodeStruct})
	reg.Register(Descriptor{Kind: KindInterface, Labels: []string{"Interface"}, Decoder: decodeInterface})
	reg.Register(Descriptor{Kind: KindType, Labels: []string{"Type"}, Decoder: decodeType})

	// controle de fluxo
	reg.Register(Descriptor{Kind: KindIfNode, Labels: []string{"IfNode"}, Decoder: decodeIfNode})
	reg.Register(Descriptor{Kind: KindSwitchNode, Labels: []string{"SwitchNode"}, Decoder: decodeSwitchNode})
	reg.Register(Descriptor{Kind: KindLoopNode, Labels: []string{"LoopNode"}, Decoder: decodeLoopNode})
	reg.Register(Descriptor{Kind: KindTryNode, Labels: []string{"TryNode"}, Decoder: decodeTryNode})
	reg.Register(Descriptor{Kind: KindBlock, Labels: []string{"Block"}, Decoder: decodeBlock})
	reg.Register(Descriptor{Kind: KindReturn, Labels: []string{"ReturnNode"}, Decoder: decodeReturnNode})
	reg.Register(Descriptor{Kind: KindThrow, Labels: []string{"ThrowNode"}, Decoder: decodeThrowNode})

	// tech (globais)
	reg.Register(Descriptor{Kind: KindLanguage, Labels: []string{"Language"}, Decoder: decodeLanguage})
	reg.Register(Descriptor{Kind: KindFramework, Labels: []string{"Framework"}, Decoder: decodeFramework})
	reg.Register(Descriptor{Kind: KindRuntime, Labels: []string{"Runtime"}, Decoder: decodeRuntime})
	reg.Register(Descriptor{Kind: KindErrorType, Labels: []string{"ErrorType"}, Decoder: decodeErrorType})
	reg.Register(Descriptor{Kind: KindLogNode, Labels: []string{"LogNode"}, Decoder: decodeLogNode})
	reg.Register(Descriptor{Kind: KindConfig, Labels: []string{"Config"}, Decoder: decodeConfig})
	reg.Register(Descriptor{Kind: KindMiddleware, Labels: []string{"Middleware"}, Decoder: decodeMiddleware})
	reg.Register(Descriptor{Kind: KindEntry, Labels: []string{"EntryNode"}, Decoder: decodeEntryNode})
	reg.Register(Descriptor{Kind: KindExit, Labels: []string{"ExitNode"}, Decoder: decodeExitNode})
	reg.Register(Descriptor{Kind: KindJoin, Labels: []string{"JoinNode"}, Decoder: decodeJoinNode})

	// chamadas — expansíveis
	reg.Register(Descriptor{
		Kind:    KindCallFunction,
		Labels:  []string{"CallFunction"},
		Decoder: decodeCallFunction,
		// Expansion agora segue EdgeExpandsTo (Phase 2.5) — separa a
		// affordance de UI da semântica de CALLS.
		Expansion: ExpansionRule{FollowEdge: EdgeExpandsTo, Depth: 1},
	})
	reg.Register(Descriptor{
		Kind:      KindCallHTTP,
		Labels:    []string{"CallHTTP"},
		Decoder:   decodeCallHTTP,
		Expansion: ExpansionRule{FollowEdge: EdgeExpandsTo, Depth: 1},
	})
	reg.Register(Descriptor{
		Kind:    KindCallDB,
		Labels:  []string{"CallDB"},
		Decoder: decodeCallDB,
		// CallDB não expande — não há corpo do "outro lado".
	})
	reg.Register(Descriptor{
		Kind:    KindPublish,
		Labels:  []string{"PublishEvent"},
		Decoder: decodePublish,
	})
	reg.Register(Descriptor{
		Kind:      KindConsume,
		Labels:    []string{"ConsumeEvent"},
		Decoder:   decodeConsume,
		Expansion: ExpansionRule{FollowEdge: EdgeContains, Depth: 1},
	})
}

// DecodeByKind é o ponto de entrada usado pelo mapper: dado kind + props,
// devolve o Node concreto. Erro se o kind não estiver registrado.
func DecodeByKind(kind Kind, props map[string]any) (Node, error) {
	d, ok := DefaultRegistry.Get(kind)
	if !ok {
		return nil, fmt.Errorf("unknown kind %q — not registered in DefaultRegistry", kind)
	}
	return d.Decoder(props)
}
