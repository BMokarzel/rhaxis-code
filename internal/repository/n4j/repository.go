package n4j

import (
	"context"
	"errors"

	"github.com/BMokarzel/rhaxis-code.git/internal/entity/node"
	logger "github.com/BMokarzel/rhaxis-code.git/pkg/log"
)

// N4J é o adapter Neo4j. Satisfaz, por duck typing, os contratos
// Repository declarados em cada módulo (read, extract, observe).
type N4J struct {
	Log *logger.Log
}

func New(log *logger.Log) *N4J {
	return &N4J{Log: log}
}

var errNotImplemented = errors.New("n4j: not implemented")

// FindNode — usado pelo módulo read.
func (r *N4J) FindNode(_ context.Context, _ string) (*node.Node, error) {
	return nil, errNotImplemented
}

// SaveNode — usado pelo módulo extract.
func (r *N4J) SaveNode(_ context.Context, _ *node.Node) error {
	return errNotImplemented
}

// Count — usado pelo módulo observe.
func (r *N4J) Count(_ context.Context, _ node.NodeType) (int64, error) {
	return 0, errNotImplemented
}
