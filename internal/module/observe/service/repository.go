package observe_service

import (
	"context"

	"github.com/BMokarzel/rhaxis-code.git/internal/entity/node"
)

type Repository interface {
	Count(ctx context.Context, kind node.NodeType) (int64, error)
}
