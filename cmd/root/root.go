package root

import (
	"context"

	"github.com/BMokarzel/rhaxis-code.git/cmd/root/serve"
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	c := &cobra.Command{
		Use:           "rhaxis",
		Short:         "Rhaxis CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	c.AddCommand(serve.NewServeCmd())
	return c
}

// Execute roda o comando raiz propagando o ctx (com sinais) para os subcomandos.
func Execute(ctx context.Context) error {
	return newRootCmd().ExecuteContext(ctx)
}
