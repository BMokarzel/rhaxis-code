// Package loader consome o payload JSON emitido pelo extrator TS (ou qualquer
// outro que respeite docs/design-codegraph.md) e escreve no codegraph via
// repository.Writer. Não fala Neo4j diretamente — a Writer é o único contrato.
//
// Idempotente: rodar de novo em cima do mesmo payload não gera duplicatas
// (MERGE-based no Writer).
package loader

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Payload é a raiz do JSON produzido pelo extrator. Nomes espelham
// extractor/schema/extraction.schema.json.
type Payload struct {
	SchemaVersion string      `json:"schemaVersion"`
	ExtractedAt   string      `json:"extractedAt"`
	SourceRev     string      `json:"sourceRev,omitempty"`
	Service       ServiceMeta `json:"service"`
	Nodes         []NodeIn    `json:"nodes"`
	Edges         []EdgeIn    `json:"edges"`
}

// ServiceMeta é o resumo do serviço — o node Service em si aparece também
// em Payload.Nodes com kind=Service. Este bloco é conveniência p/ ferramentas
// que querem o header sem varrer nodes.
type ServiceMeta struct {
	URN       string `json:"urn"`
	Name      string `json:"name"`
	Language  string `json:"language"`
	Framework string `json:"framework,omitempty"`
	Runtime   string `json:"runtime,omitempty"`
	RepoURL   string `json:"repoURL,omitempty"`
}

// NodeIn é o formato bruto de um node no JSON. Properties é kind-specific;
// o loader achata para o formato que domain.DecodeByKind espera.
type NodeIn struct {
	URN        string         `json:"urn"`
	Kind       string         `json:"kind"`
	Name       string         `json:"name"`
	Resolved   *bool          `json:"resolved,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

// EdgeIn é o formato bruto de uma edge no JSON.
type EdgeIn struct {
	From  string         `json:"from"`
	To    string         `json:"to"`
	Type  string         `json:"type"`
	Props map[string]any `json:"props,omitempty"`
}

// ReadPayload lê e valida a versão do schema. Não faz validação semântica
// profunda — isso fica para o LoadInto/Writer.
func ReadPayload(r io.Reader) (*Payload, error) {
	var p Payload
	dec := json.NewDecoder(r)
	dec.UseNumber() // preserva ints em props (index) sem virar float64
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("loader: parse payload: %w", err)
	}
	if p.SchemaVersion != "1" {
		return nil, fmt.Errorf("loader: unsupported schemaVersion %q (want \"1\")", p.SchemaVersion)
	}
	// Normaliza json.Number → int64 / float64 recursivamente. O driver Neo4j
	// aceita int64/float64 nativos mas não json.Number.
	for i := range p.Nodes {
		p.Nodes[i].Properties = normalizeNumbers(p.Nodes[i].Properties)
		p.Nodes[i].Metadata = normalizeNumbers(p.Nodes[i].Metadata)
	}
	for i := range p.Edges {
		p.Edges[i].Props = normalizeNumbers(p.Edges[i].Props)
	}
	return &p, nil
}

// ReadPayloadFile é açúcar para o CLI.
func ReadPayloadFile(path string) (*Payload, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("loader: open %s: %w", path, err)
	}
	defer f.Close()
	return ReadPayload(f)
}

// normalizeNumbers converte json.Number → int64 quando possível, senão float64.
// Necessário porque UseNumber() preserva a representação, mas o driver Neo4j
// só aceita tipos Go nativos.
func normalizeNumbers(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	for k, v := range m {
		switch t := v.(type) {
		case json.Number:
			if i, err := t.Int64(); err == nil {
				m[k] = i
			} else if f, err := t.Float64(); err == nil {
				m[k] = f
			} else {
				m[k] = t.String()
			}
		case map[string]any:
			m[k] = normalizeNumbers(t)
		case []any:
			for i, elem := range t {
				if n, ok := elem.(json.Number); ok {
					if iv, err := n.Int64(); err == nil {
						t[i] = iv
					} else if fv, err := n.Float64(); err == nil {
						t[i] = fv
					}
				}
			}
		}
	}
	return m
}

