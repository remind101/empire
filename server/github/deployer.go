package github

import (
	"io"

	"github.com/ejholmes/hookshot/events"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/tugboat"
	"golang.org/x/net/context"
	"golang.org/x/net/trace"
)

// Deployer represents something that can deploy a github deployment.
type Deployer interface {
	// Deploy performs the deployment, writing output to w.
	Deploy(context.Context, events.Deployment, io.Writer) error
}

type DeployerFunc func(context.Context, events.Deployment, io.Writer) error

func (fn DeployerFunc) Deploy(ctx context.Context, event events.Deployment, w io.Writer) error {
	return fn(ctx, event, w)
}

// empire mocks the Empire interface we use.
type empireClient interface {
	Deploy(context.Context, empire.DeploymentsCreateOpts) (*empire.Release, error)
}

// EmpireDeployer is a deployer implementation that uses the Deploy method in
// Empire to perform the deployment.
type EmpireDeployer struct {
	empire empireClient
	ImageBuilder
}

// NewEmpireDeployer returns a new EmpireDeployer instance.
func NewEmpireDeployer(e *empire.Empire) *EmpireDeployer {
	return &EmpireDeployer{
		empire: e,
	}
}

// Deploy builds/determines the docker image to deploy, then deploys it with
// Empire.
func (d *EmpireDeployer) Deploy(ctx context.Context, event events.Deployment, w io.Writer) error {
	img, err := d.BuildImage(ctx, event)
	if err != nil {
		return err
	}

	_, err = d.empire.Deploy(ctx, empire.DeploymentsCreateOpts{
		Image:  img,
		Output: w,
		User:   &empire.User{Name: event.Deployment.Creator.Login},
	})

	return err
}

// TugboatDeployer is an implementtion of the deployer interface that sends logs
// and updates the status of the deployment within a Tugboat instance.
type TugboatDeployer struct {
	deployer Deployer
	client   *tugboat.Client
}

// NotifyTugboat wraps a Deployer to sent deployment logs and status updates to
// a Tugboat instance.
func NotifyTugboat(d Deployer, url string) *TugboatDeployer {
	c := tugboat.NewClient(nil)
	c.URL = url
	return &TugboatDeployer{
		deployer: d,
		client:   c,
	}
}

func (d *TugboatDeployer) Deploy(ctx context.Context, event events.Deployment, out io.Writer) error {
	opts := tugboat.NewDeployOptsFromEvent(event)

	// Perform the deployment, wrapped in Deploy. This will automatically
	// write hte logs to tugboat and update the deployment status when this
	// function returns.
	_, err := d.client.Deploy(ctx, opts, provider(func(ctx context.Context, _ *tugboat.Deployment, w io.Writer) error {
		// What we send to tugboat should be a plain text stream.
		p := dockerutil.DecodeJSONMessageStream(w)

		// Write logs to both tugboat as well as the writer we were
		// provided (probably stdout).
		w = io.MultiWriter(p, out)

		if err := d.deployer.Deploy(ctx, event, w); err != nil {
			return err
		}

		if err := p.Err(); err != nil {
			return err
		}

		return nil
	}))

	return err
}

// provider implements the tugboat.Provider interface.
type provider func(context.Context, *tugboat.Deployment, io.Writer) error

func (fn provider) Name() string {
	return "empire"
}

func (fn provider) Deploy(ctx context.Context, d *tugboat.Deployment, w io.Writer) error {
	return fn(ctx, d, w)
}

// DeployAsync wraps a Deployer to perform the Deploy within a goroutine.
func DeployAsync(d Deployer) Deployer {
	return DeployerFunc(func(ctx context.Context, event events.Deployment, w io.Writer) error {
		go d.Deploy(ctx, event, w)
		return nil
	})
}

// TraceDeploy wraps a Deployer to perform tracing with package trace.
func TraceDeploy(d Deployer) Deployer {
	return DeployerFunc(func(ctx context.Context, event events.Deployment, w io.Writer) (err error) {
		if tr, ok := trace.FromContext(ctx); ok {
			tr.LazyPrintf("Starting GitHub Deployment")
		}
		err = d.Deploy(ctx, event, w)
		if tr, ok := trace.FromContext(ctx); ok {
			tr.LazyPrintf("Finished GitHub Deployment")
			if err != nil {
				tr.SetError()
			}
		}
		return err
	})
}
