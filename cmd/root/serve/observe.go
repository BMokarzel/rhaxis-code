package serve

import (
	"github.com/BMokarzel/rhaxis-code.git/config"
	"github.com/BMokarzel/rhaxis-code.git/internal/module/observe"
	pkg_http "github.com/BMokarzel/rhaxis-code.git/pkg/http"
	"github.com/spf13/cobra"
)

func newObserveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "observe",
		Short: "Sobe o módulo Observe",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServer(cmd, func(cfg *config.Config) pkg_http.Server {
				return observe.NewServer(cfg)
			})
		},
	}
}
