package observe_service

import (
	"context"

	"github.com/BMokarzel/rhaxis-code.git/internal/entity/node"
	logger "github.com/BMokarzel/rhaxis-code.git/pkg/log"
)

// Service expõe métricas / telemetria do grafo.
type Service interface {
	Stats(ctx context.Context) (map[node.NodeType]int64, error)
}

type service struct {
	log  *logger.Log
	repo Repository
}

func New(log *logger.Log, repo Repository) Service {
	return &service{log: log, repo: repo}
}

var trackedKinds = []node.NodeType{
	node.ServiceNode, node.ModuleNode, node.EndpointNode,
	node.FunctionNode, node.DataNode, node.VariableNode,
	node.CallNode, node.SchemaNode,
}

func (s *service) Stats(ctx context.Context) (map[node.NodeType]int64, error) {
	out := make(map[node.NodeType]int64, len(trackedKinds))
	for _, k := range trackedKinds {
		c, err := s.repo.Count(ctx, k)
		if err != nil {
			return nil, err
		}
		out[k] = c
	}
	return out, nil
}
