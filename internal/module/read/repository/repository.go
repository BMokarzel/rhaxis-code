package read_repository

import (
	"context"

	"github.com/BMokarzel/rhaxis-code.git/internal/entity/node"
)

// Repository é a porta de dados consumida pelo Service do módulo read.
// O adapter concreto (n4j, memory) é injetado em tempo de wiring.
type Repository interface {
	FindNode(ctx context.Context, id string) (*node.Node, error)
}
