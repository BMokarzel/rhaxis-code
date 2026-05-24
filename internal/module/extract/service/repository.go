package extract_service

import (
	"context"

	"github.com/BMokarzel/rhaxis-code.git/internal/entity/node"
)

type Repository interface {
	SaveNode(ctx context.Context, n *node.Node) error
}
