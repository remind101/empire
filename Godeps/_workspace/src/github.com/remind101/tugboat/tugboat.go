package tugboat

import (
	"errors"
	"fmt"
	"io"

	"code.google.com/p/goauth2/oauth"
	"github.com/google/go-github/github"
	"github.com/joshk/pusher"
	"github.com/mattes/migrate/migrate"
	"golang.org/x/net/context"
)

// BaseURL is the baseURL where tugboat is running.
var BaseURL string

// Config is configuration for a new Tugboat instance.
type Config struct {
	Pusher struct {
		URL string
	}

	GitHub struct {
		Token string
	}

	// ProviderSecret is the secret used to sign JWT tokens for
	// authentication external providers.
	ProviderSecret []byte

	// DB connection string.
	DB string
}

// Tugboat provides methods for performing deployments.
type Tugboat struct {
	// Provider is a provider that will be used to fullfill deployments.
	Providers []Provider

	// MatchEnvironment controls what environments are allowed to be
	// deployed to. If a value is provided, Tugboat will only deploy if the
	// environment matches.
	MatchEnvironment string

	store *store

	deployments deploymentsService
	logs        logsService
	tokens      tokensService
}

// New returns a new Tugboat instance.
func New(config Config) (*Tugboat, error) {
	db, err := dialDB(config.DB)
	if err != nil {
		return nil, err
	}
	store := &store{db: db}

	pusher, err := newPusherClient(config.Pusher.URL)
	if err != nil {
		return nil, err
	}

	github := newGitHubClient(config.GitHub.Token)

	var updater multiUpdater

	if config.Pusher.URL != "" {
		updater = append(updater, &pusherUpdater{
			pusher: pusher,
		})
	}

	if config.GitHub.Token != "" {
		updater = append(updater, &githubUpdater{
			github: github.Repositories,
		})
	}

	deployments := newDeploymentsService(store, updater)
	logs := newLogsService(store, pusher)
	tokens := newTokensService(config.ProviderSecret)

	return &Tugboat{
		store:       store,
		deployments: deployments,
		logs:        logs,
		tokens:      tokens,
	}, nil
}

// TokensCreate creates a new authentication token for an external provider.
func (t *Tugboat) TokensCreate(token *Token) error {
	return t.tokens.TokensCreate(token)
}

// TokensFind finds a provider authentication token.
func (t *Tugboat) TokensFind(token string) (*Token, error) {
	return t.tokens.TokensFind(token)
}

// Deployments returns the most recent Deployments.
func (t *Tugboat) Deployments(q DeploymentsQuery) ([]*Deployment, error) {
	return t.store.Deployments(q)
}

// DeploymentsFind finds a Deployment by id.
func (t *Tugboat) DeploymentsFind(id string) (*Deployment, error) {
	return t.store.DeploymentsFind(id)
}

// DeploymentsCreate creates a new Deployment.
func (t *Tugboat) DeploymentsCreate(opts DeployOpts) (*Deployment, error) {
	d := newDeployment(opts)
	d.Started(opts.Provider)
	return d, t.deployments.DeploymentsCreate(d)
}

// DeploymentsUpdate updates a Deployment.
func (t *Tugboat) DeploymentsUpdate(d *Deployment) error {
	return t.deployments.DeploymentsUpdate(d)
}

// UpdateStatus updates the deployment using the given StatusUpdate
func (t *Tugboat) UpdateStatus(d *Deployment, update StatusUpdate) error {
	switch update.Status {
	case StatusFailed:
		d.Failed()
	case StatusErrored:
		var err error
		if update.Error != nil {
			err = *update.Error
		} else {
			err = errors.New("no error provided")
		}
		d.Errored(err)
	case StatusSucceeded:
		d.Succeeded()
	default:
		return errors.New("invalid status")
	}

	return t.DeploymentsUpdate(d)
}

// WriteLogs reads each line from r and creates a log line for the Deployment.
func (t *Tugboat) WriteLogs(d *Deployment, r io.Reader) error {
	w := &logWriter{
		createLogLine: t.logs.LogLinesCreate,
		deploymentID:  d.ID,
	}

	_, err := io.Copy(w, r)
	return err
}

// Logs returns a single string of text for all the entire log stream.
// TODO: Make this into something that writes to an io.Writer and change the
// frontend to stream the logs from a streaming endpoint.
func (t *Tugboat) Logs(d *Deployment) (string, error) {
	lines, err := t.store.LogLines(d)
	if err != nil {
		return "", err
	}

	out := ""
	for _, line := range lines {
		out += line.Text
	}

	return out, nil
}

// DEPRECATED: Deploy triggers a new deployment.
func (t *Tugboat) Deploy(ctx context.Context, opts DeployOpts) ([]*Deployment, error) {
	if t.MatchEnvironment != "" {
		if t.MatchEnvironment != opts.Environment {
			return nil, nil
		}
	}

	ps := t.Providers
	if len(ps) == 0 {
		ps = []Provider{&NullProvider{}}
	}

	var deployments []*Deployment

	for _, p := range ps {
		d, err := deploy(ctx, opts, p, t)
		if err != nil {
			return nil, err
		}

		deployments = append(deployments, d)
	}

	return deployments, nil
}

func (t *Tugboat) Reset() error {
	return t.store.Reset()
}

type client interface {
	DeploymentsCreate(DeployOpts) (*Deployment, error)
	WriteLogs(*Deployment, io.Reader) error
	UpdateStatus(*Deployment, StatusUpdate) error
}

// Deploy is the primary business logic for performing a deployment. It:
//
// 1. Creates the deployment within Tugboat.
// 2. Performs the deployment using the Provider.
// 3. Writes the log output to the deployment.
// 4. Updates the status depending on the eror returned from `fn`.
func deploy(ctx context.Context, opts DeployOpts, p Provider, t client) (deployment *Deployment, err error) {
	opts.Provider = p.Name()

	deployment, err = t.DeploymentsCreate(opts)
	if err != nil {
		return
	}

	r, w := io.Pipe()

	logsDone := make(chan struct{}, 1)
	go func() {
		if werr := t.WriteLogs(deployment, r); werr != nil {
			// If there was an error writing the logs, close the
			// Write side of the pipe.
			r.CloseWithError(err)
		}
		close(logsDone)
	}()

	t.UpdateStatus(deployment, statusUpdate(func() error {
		err = p.Deploy(ctx, deployment, w)
		return err
	}))

	w.Close()

	// Wait for logs to finish streaming.
	<-logsDone

	return
}

// statusUpdate calls fn and, depending on the error or panic, returns an
// appropriate StatusUpdate.
func statusUpdate(fn func() error) (update StatusUpdate) {
	var err error

	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("%v", v)

			if v, ok := v.(error); ok {
				err = v
			}
		}

		// No error or panic. Success!
		if err == nil {
			update.Status = StatusSucceeded
			return
		}

		// ErrFailed is returned when the deployment failed and the user
		// should check the deployment logs.
		if err == ErrFailed {
			update.Status = StatusFailed
		} else {
			update.Status = StatusErrored
			update.Error = &err
		}
	}()

	err = fn()

	return
}

// Migrate runs the migrations.
func Migrate(db, path string) ([]error, bool) {
	return migrate.UpSync(db, path)
}

func newPusherClient(uri string) (Pusher, error) {
	if uri == "" {
		return &nullPusher{}, nil
	}

	c, err := ParsePusherCredentials(uri)
	if err != nil {
		return nil, err
	}

	return newAsyncPusher(
		pusher.NewClient(c.AppID, c.Key, c.Secret),
		1000,
	), nil
}

func newGitHubClient(token string) *github.Client {
	t := &oauth.Transport{
		Token: &oauth.Token{AccessToken: token},
	}

	return github.NewClient(t.Client())
}
