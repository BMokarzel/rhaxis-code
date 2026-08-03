package neo4j

import (
	"fmt"

	"github.com/BMokarzel/rhaxis-code.git/domain"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

// nodeToDomain traduz um neo4j.Node em domain.Node, delegando ao Decoder
// registrado no DefaultRegistry pelo kind da property.
//
// A escolha de discriminar por property `kind` (e não pela label) é
// intencional: labels são um vetor, e queries futuras que retornam nodes
// heterogêneos ficam mais simples se o discriminador for uma prop única.
// A label continua sendo o índice de leitura no Neo4j.
func nodeToDomain(n dbtype.Node) (domain.Node, error) {
	if n.Props == nil {
		return nil, fmt.Errorf("neo4j node has nil properties")
	}
	kindRaw, ok := n.Props["kind"].(string)
	if !ok {
		return nil, fmt.Errorf("neo4j node missing string 'kind' property (labels=%v)", n.Labels)
	}
	return domain.DecodeByKind(domain.Kind(kindRaw), n.Props)
}

// nodeToDomainSafe é a variante que absorve erro em log-only mode. Retorna
// (nil, false) quando o node não puder ser decodado. O reader usa isso para
// aplicar a política do §4: "erros de decode omitem o node, não derrubam a
// query".
func nodeToDomainSafe(n dbtype.Node) (domain.Node, bool) {
	dn, err := nodeToDomain(n)
	if err != nil {
		// TODO: injetar logger; por ora, silenciar é intencional e documentado.
		return nil, false
	}
	return dn, true
}

// asNeoNode faz cast seguro de um valor genérico do driver para dbtype.Node.
// Retorna (zero, false) se o valor for nil ou não for Node.
func asNeoNode(v any) (dbtype.Node, bool) {
	if v == nil {
		return dbtype.Node{}, false
	}
	n, ok := v.(dbtype.Node)
	return n, ok
}

// asIntProp lê um índice/inteiro genérico dentro de um record record.
func asIntProp(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	}
	return 0
}

// asStringProp lê uma string genérica.
func asStringProp(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// asListProp lê uma lista genérica.
func asListProp(v any) []any {
	if l, ok := v.([]any); ok {
		return l
	}
	return nil
}

// asMapProp lê um mapa genérico (rows do driver vêm como map[string]any).
func asMapProp(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}
