package tugboat

import (
	"errors"
	"fmt"
	"io"

	"golang.org/x/net/context"
)

var (
	// ErrFailed can be used by providers to indicate that the deployment
	// failed.
	ErrFailed = errors.New("deployment failed")
)

// Provider is something that's capable of fullfilling a deployment.
type Provider interface {
	// Deploy should perform the deployment. An io.Writer can be provided
	// for providers to write log output to. If the deployment failed
	// without a specific error, and the user should view the logs to find
	// out why, then an ErrFailed should be returned.
	Deploy(context.Context, *Deployment, io.Writer) error

	// Name should return the name of this provider.
	Name() string
}

type ProviderFunc func(context.Context, *Deployment, io.Writer) error

func (f ProviderFunc) Deploy(ctx context.Context, d *Deployment, w io.Writer) error {
	return f(ctx, d, w)
}

func (f ProviderFunc) Name() string {
	return fmt.Sprintf("func: %v", f)
}

var _ Provider = &NullProvider{}

// NullProvider is a Provider that does nothing.
type NullProvider struct{}

func (p *NullProvider) Deploy(ctx context.Context, d *Deployment, w io.Writer) error {
	return nil
}

func (p *NullProvider) Name() string {
	return "null"
}
