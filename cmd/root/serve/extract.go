package serve

import (
	"github.com/BMokarzel/rhaxis-code.git/config"
	"github.com/BMokarzel/rhaxis-code.git/internal/module/extract"
	pkg_http "github.com/BMokarzel/rhaxis-code.git/pkg/http"
	"github.com/spf13/cobra"
)

func newExtractCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "extract",
		Short: "Sobe o módulo Extract",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServer(cmd, func(cfg *config.Config) pkg_http.Server {
				return extract.NewServer(cfg)
			})
		},
	}
}
