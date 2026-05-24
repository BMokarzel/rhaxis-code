package serve

import (
	"github.com/BMokarzel/rhaxis-code.git/config"
	pkg_http "github.com/BMokarzel/rhaxis-code.git/pkg/http"
	"github.com/spf13/cobra"
)

// NewServeCmd agrupa os subcomandos que sobem um módulo HTTP por vez.
func NewServeCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "serve",
		Short: "Sobe o servidor de um módulo (read | extract | observe)",
	}
	c.AddCommand(newReadCmd(), newExtractCmd(), newObserveCmd())
	return c
}

// runServer é o esqueleto compartilhado: carrega config e roda o Server
// com graceful shutdown usando o ctx do comando raiz.
func runServer(cmd *cobra.Command, build func(*config.Config) pkg_http.Server) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	srv := build(cfg)
	timeouts := pkg_http.Timeouts{
		Read:     cfg.HTTP.ReadTimeout,
		Write:    cfg.HTTP.WriteTimeout,
		Idle:     cfg.HTTP.IdleTimeout,
		Shutdown: cfg.HTTP.ShutdownTimeout,
	}
	cmd.Printf("starting %s on %s\n", srv.Name(), srv.Addr())
	return pkg_http.Run(cmd.Context(), srv, timeouts)
}
