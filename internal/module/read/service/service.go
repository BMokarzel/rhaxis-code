package read_service

import (
	"context"

	"github.com/BMokarzel/rhaxis-code.git/internal/entity/node"
	read_repository "github.com/BMokarzel/rhaxis-code.git/internal/module/read/repository"
	logger "github.com/BMokarzel/rhaxis-code.git/pkg/log"
)

// Service expõe as operações de leitura do grafo.
type Service interface {
	GetNode(ctx context.Context, id string) (*node.Node, error)
}

type service struct {
	log  *logger.Log
	repo read_repository.Repository
}

func New(log *logger.Log, repo read_repository.Repository) Service {
	return &service{log: log, repo: repo}
}

func (s *service) GetNode(ctx context.Context, id string) (*node.Node, error) {
	return s.repo.FindNode(ctx, id)
}
