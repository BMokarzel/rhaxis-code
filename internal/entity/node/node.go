package node

import "time"

type URN string

type NodeType string

const (
	ServiceNode    NodeType = "service"  // raiz de manifest (go.mod, package.json, pyproject.toml…)
	ModuleNode     NodeType = "module"   // pasta/namespace agregador (F-018, CONTAINS aninhável)
	EndpointNode   NodeType = "endpoint" // handler HTTP (método+rota)
	FunctionNode   NodeType = "function" // função/método declarado
	DataNode       NodeType = "type"     // struct/class/interface/enum/alias/union (F-021)
	VariableNode   NodeType = "variable" // package-level var/const (F-021)
	CallNode       NodeType = "call"     // call site nó (família F-019/F-020; subkind em Call.Kind)
	SchemaNode     NodeType = "schema"   // contrato externo (proto, OpenAPI) — F-022
	HttpMethodNode NodeType = "http_method"
)

type ExtractionMethod string

const (
	API      ExtractionMethod = "api"
	Inferred ExtractionMethod = "inferred"
	Declared ExtractionMethod = "declared"
	Imported ExtractionMethod = "imported"
)

// Lineage descreve a cadeia causal de um fato inferido/derivado.
type Lineage struct {
	DerivedFrom []URN  `json:"derived_from,omitempty"`
	Rule        string `json:"rule,omitempty"`
}

// Meta carrega os atributos transversais presentes em todo nó.
//
// Bitemporal:
//   - ValidFrom/ValidTo  → quando o fato vale no mundo real.
//   - ObservedAt         → quando o sistema soube (transaction time).
type Meta struct {
	Version          uint64           `json:"version"`
	ValidFrom        time.Time        `json:"valid_from"`
	ValidTo          *time.Time       `json:"valid_to,omitempty"`
	ObservedAt       time.Time        `json:"observed_at"`
	ExtractionMethod ExtractionMethod `json:"extraction_method"`
	Confidence       float32          `json:"confidence"`
	Labels           []string         `json:"labels,omitempty"`
	Properties       map[string]any   `json:"properties,omitempty"`
	Lineage          Lineage          `json:"lineage,omitempty"`
}

// IsCurrent retorna true se o fato ainda é o corrente (sem ValidTo).
func (m Meta) IsCurrent() bool { return m.ValidTo == nil }

type NodeInterface interface {
	NodeName() string
	NodeURN() URN
	NodeType() NodeType
	NodeMeta() Meta
}

type Node struct {
	Name string
	URN  URN
	Type NodeType
	Meta Meta
}

func (n Node) NodeName() string   { return n.Name }
func (n Node) NodeURN() URN       { return n.URN }
func (n Node) NodeType() NodeType { return n.Type }
func (n Node) NodeMeta() Meta     { return n.Meta }
