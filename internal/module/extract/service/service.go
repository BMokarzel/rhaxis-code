package extract_service

import (
	"context"

	"github.com/BMokarzel/rhaxis-code.git/internal/entity/node"
	logger "github.com/BMokarzel/rhaxis-code.git/pkg/log"
)

// Service expõe as operações de ingestão (parse → grafo).
type Service interface {
	Ingest(ctx context.Context, n *node.Node) error
}

type service struct {
	log  *logger.Log
	repo Repository
}

func New(log *logger.Log, repo Repository) Service {
	return &service{log: log, repo: repo}
}

func (s *service) Ingest(ctx context.Context, n *node.Node) error {
	return s.repo.SaveNode(ctx, n)
}
