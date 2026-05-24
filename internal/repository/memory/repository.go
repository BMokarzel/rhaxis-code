package memory

import (
	"context"
	"sync"

	"github.com/BMokarzel/rhaxis-code.git/internal/entity/node"
	logger "github.com/BMokarzel/rhaxis-code.git/pkg/log"
)

// Memory é um adapter in-memory útil para testes e desenvolvimento.
// Satisfaz os mesmos contratos que N4J.
type Memory struct {
	Log *logger.Log

	mu    sync.RWMutex
	nodes map[string]*node.Node
}

func New(log *logger.Log) *Memory {
	return &Memory{Log: log, nodes: make(map[string]*node.Node)}
}

func (r *Memory) FindNode(_ context.Context, id string) (*node.Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if n, ok := r.nodes[id]; ok {
		return n, nil
	}
	return nil, nil
}

func (r *Memory) SaveNode(_ context.Context, _ *node.Node) error {
	// id ainda não existe em node.Node; quando existir, persistir aqui.
	return nil
}

func (r *Memory) Count(_ context.Context, _ node.NodeType) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.nodes)), nil
}
