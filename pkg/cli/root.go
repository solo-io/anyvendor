package cli

import (
	"context"

	"github.com/solo-io/protodep/pkg/cli/internal/cmd/ensure"
	"github.com/solo-io/protodep/pkg/cli/internal/flags"
	"github.com/solo-io/protodep/pkg/cli/internal/options"
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	opts := &options.Options{
		Ctx: context.Background(),
	}
	cmd := &cobra.Command{
		Use:   "protodep",
		Short: "",
		Long:  "",
	}

	cmd.AddCommand(
		ensure.Cmd(opts),
	)

	flags.RootFlags(&opts.Root, cmd.PersistentFlags())

	return cmd
}
