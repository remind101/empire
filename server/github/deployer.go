package github

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ejholmes/hookshot/events"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/dockerutil"
	streamhttp "github.com/remind101/empire/pkg/stream/http"
	"github.com/remind101/pkg/trace"
	"github.com/remind101/tugboat"
	netcontext "golang.org/x/net/context"
)

// Deployer represents something that can deploy a github deployment.
type Deployer interface {
	// Deploy performs the deployment, writing output to w. The output
	// written to w should be plain text.
	Deploy(context.Context, events.Deployment, io.Writer) error
}

// DeployerFunc is a function that implements the Deployer interface.
type DeployerFunc func(context.Context, events.Deployment, io.Writer) error

func (fn DeployerFunc) Deploy(ctx context.Context, event events.Deployment, w io.Writer) error {
	return fn(ctx, event, w)
}

// empire mocks the Empire interface we use.
type empireClient interface {
	Deploy(context.Context, empire.DeployOpts) (*empire.Release, error)
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
	img, err := d.BuildImage(ctx, w, event)
	if err != nil {
		return err
	}

	// What we write to w should be plain text. `p` will get the jsonmessage
	// stream.
	p := dockerutil.DecodeJSONMessageStream(w)

	message := event.Deployment.Description
	if message == "" {
		message = fmt.Sprintf("GitHub deployment #%d of %s", event.Deployment.ID, event.Repository.FullName)
	}
	_, err = d.empire.Deploy(ctx, empire.DeployOpts{
		Image:   img,
		Output:  empire.NewDeploymentStream(p),
		User:    &empire.User{Name: event.Deployment.Creator.Login},
		Stream:  true,
		Message: message,
	})
	if err != nil {
		return err
	}

	return p.Err()
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
	_, err := d.client.Deploy(ctx, opts, provider(func(ctx netcontext.Context, _ *tugboat.Deployment, w io.Writer) error {
		defer close(streamhttp.Heartbeat(w, 10*time.Second))

		// Write logs to both tugboat as well as the writer we were
		// provided (probably stdout).
		w = io.MultiWriter(w, out)

		return d.deployer.Deploy(ctx, event, w)
	}))

	return err
}

// provider implements the tugboat.Provider interface.
type provider func(netcontext.Context, *tugboat.Deployment, io.Writer) error

func (fn provider) Name() string {
	return "empire"
}

func (fn provider) Deploy(ctx netcontext.Context, d *tugboat.Deployment, w io.Writer) error {
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
		ctx, done := trace.Trace(ctx)
		err = d.Deploy(ctx, event, w)
		done(err, "Deploy",
			"repository", event.Repository.FullName,
			"creator", event.Deployment.Creator.Login,
			"ref", event.Deployment.Ref,
			"sha", event.Deployment.Sha,
		)
		return err
	})
}
