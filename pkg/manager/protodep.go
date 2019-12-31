package manager

import (
	"context"

	"github.com/solo-io/anyvendor/anyvendor"
)

/*
	An internal only interface used to represent the different types of available sources
	for non-go vendored files.
*/
type depFactory interface {
	Ensure(ctx context.Context, opts *anyvendor.Config) error
}

/*
	The manager is the external facing object that will be responsible for ensuring
	a given anyvendor config, as outlined by the `anyvendor.Config` object.
*/
type Manager struct {
	depFactories []depFactory
}

func NewManager(ctx context.Context, cwd string) (*Manager, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	goMod, err := NewGoModFactory(cwd)
	if err != nil {
		return nil, err
	}
	return &Manager{
		depFactories: []depFactory{
			goMod,
		},
	}, nil
}

func (m *Manager) Ensure(ctx context.Context, opts *anyvendor.Config) error {
	if err := opts.Validate(); err != nil {
		return err
	}
	for _, v := range m.depFactories {
		if err := v.Ensure(ctx, opts); err != nil {
			return err
		}
	}
	return nil
}
