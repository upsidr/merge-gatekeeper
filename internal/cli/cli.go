package cli

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

// Tease variables will be set by command line flags.
var (
	ghToken string
)

func Run(version string, args ...string) error {
	cmd := &cobra.Command{
		Use:     "jobkeeper",
		Short:   "Manage github actions job",
		Version: version,
	}
	cmd.PersistentFlags().StringVarP(&ghToken, "token", "t", "", "set github token")
	cmd.MarkPersistentFlagRequired("token")

	cmd.AddCommand(validateCmd())

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	if err := cmd.ExecuteContext(ctx); err != nil {
		return err
	}
	return nil
}
