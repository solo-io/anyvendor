package flags

import (
	"github.com/solo-io/anyvendor/pkg/cli/internal/options"
	"github.com/spf13/pflag"
)

func fileFlag(f *string, flags *pflag.FlagSet) {
	flags.StringVarP(f, "file", "f", "anyvendor.yaml", "filepath to config file")
}

func RootFlags(opts *options.Root, flags *pflag.FlagSet) {
	fileFlag(&opts.File, flags)
}
