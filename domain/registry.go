package domain

import "sync"

// Descriptor descreve estaticamente um Kind: como decodá-lo, quais labels
// carrega no Neo4j, quais edges são válidas, e se é expansível.
type Descriptor struct {
	Kind Kind

	// Labels adicionais no Neo4j, além de :Node. Normalmente uma só, igual a Kind.
	Labels []string

	// Decoder converte um record decodificado (props map) em um Node concreto.
	Decoder DecodeFunc

	// Encoder converte um Node concreto de volta em (labels, props). Usado
	// pelo Writer para serializar antes de MERGE no Neo4j. Nullable: kinds
	// que não são escritos pelo core (só pelo extrator externo) podem
	// omitir. O Writer erra se Encoder for nil.
	Encoder EncodeFunc

	// AllowedEdges* documentam as edges válidas para este kind. Usados pelo
	// writer (extrator) e por testes de invariante; o reader confia.
	AllowedEdgesOut map[EdgeType][]Kind
	AllowedEdgesIn  map[EdgeType][]Kind

	// Expansion, quando não-zero, define como o Reader devolve um ExpansionSlot
	// para nodes deste kind (ex.: CallFunction "expande" via :CALLS).
	Expansion ExpansionRule
}

// DecodeFunc converte o map de propriedades (já extraído do record Neo4j)
// no Node concreto. Erros aqui abortam apenas o node atual, não a query inteira.
type DecodeFunc func(props map[string]any) (Node, error)

// EncodeFunc é o inverso: recebe o Node concreto, devolve as labels adicionais
// (além de :Node) e o map de propriedades pronto para persistir.
// Nada de types específicos do driver aqui — só primitives e string JSON.
type EncodeFunc func(n Node) (labels []string, props map[string]any, err error)

// ExpansionRule define como um kind expansível é seguido pelo Reader.
type ExpansionRule struct {
	// FollowEdge é a edge que leva ao alvo da expansão. Zero = não expansível.
	FollowEdge EdgeType
	// Depth é sempre 1 no contrato v1 (ver §5 do doc). Campo reservado para v2.
	Depth int
}

// IsExpandable diz se este kind gera ExpansionSlot no FlowNode.
func (d Descriptor) IsExpandable() bool {
	return d.Expansion.FollowEdge != ""
}

// Registry é o índice de kinds. Preenchido no init() de cada arquivo de
// decoder e consultado pelo mapper na hora de traduzir um record em Node.
type Registry interface {
	Register(Descriptor)
	Get(Kind) (Descriptor, bool)
	All() []Descriptor
}

// --- default registry ---------------------------------------------------

// DefaultRegistry é o singleton usado pelo mapper padrão. Não é obrigatório
// usá-lo (testes podem instanciar um Registry próprio).
var DefaultRegistry Registry = newInMemoryRegistry()

type inMemoryRegistry struct {
	mu   sync.RWMutex
	data map[Kind]Descriptor
}

func newInMemoryRegistry() *inMemoryRegistry {
	return &inMemoryRegistry{data: make(map[Kind]Descriptor)}
}

func (r *inMemoryRegistry) Register(d Descriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[d.Kind] = d
}

func (r *inMemoryRegistry) Get(k Kind) (Descriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.data[k]
	return d, ok
}

func (r *inMemoryRegistry) All() []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Descriptor, 0, len(r.data))
	for _, d := range r.data {
		out = append(out, d)
	}
	return out
}
