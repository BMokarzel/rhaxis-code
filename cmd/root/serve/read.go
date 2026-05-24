package serve

import (
	"github.com/BMokarzel/rhaxis-code.git/config"
	"github.com/BMokarzel/rhaxis-code.git/internal/module/read"
	pkg_http "github.com/BMokarzel/rhaxis-code.git/pkg/http"
	"github.com/spf13/cobra"
)

func newReadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read",
		Short: "Sobe o módulo Read",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServer(cmd, func(cfg *config.Config) pkg_http.Server {
				return read.NewServer(cfg)
			})
		},
	}
}
