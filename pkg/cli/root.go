package cli

import (
	"context"

	"github.com/solo-io/anyvendor/pkg/cli/internal/cmd/ensure"
	"github.com/solo-io/anyvendor/pkg/cli/internal/flags"
	"github.com/solo-io/anyvendor/pkg/cli/internal/options"
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	opts := &options.Options{
		Ctx: context.Background(),
	}
	cmd := &cobra.Command{
		Use:   "anyvendor",
		Short: "",
		Long:  "",
	}

	cmd.AddCommand(
		ensure.Cmd(opts),
	)

	flags.RootFlags(&opts.Root, cmd.PersistentFlags())

	return cmd
}
