package ensure

import (
	"github.com/solo-io/anyvendor/pkg/cli/internal/options"
	"github.com/spf13/cobra"
)

func Cmd(options *options.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ensure",
		Aliases: []string{"e"},
		Short:   "",
		Long:    "",
		Example: "",
	}
	return cmd
}
