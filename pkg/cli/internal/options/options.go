package options

import (
	"context"
)

type Options struct {
	Ctx  context.Context
	Root Root
}

type Root struct {
	File string
}
