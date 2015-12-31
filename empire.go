package empire // import "github.com/remind101/empire"

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/fsouza/go-dockerclient"
	"github.com/inconshreveable/log15"
	"github.com/jinzhu/gorm"
	"github.com/mattes/migrate/migrate"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/procfile"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

var (
	// DefaultOptions is a default Options instance that can be passed when
	// intializing a new Empire.
	DefaultOptions = Options{}

	// DefaultReporter is the default reporter.Reporter to use.
	DefaultReporter = reporter.NewLogReporter()
)

const (
	// WebPort is the default PORT to set on web processes.
	WebPort = 8080

	// WebProcessType is the process type we assume are web server processes.
	WebProcessType = "web"
)

// ProcfileExtractor is a function that can extract a Procfile from an image.
type ProcfileExtractor func(context.Context, image.Image, io.Writer) (procfile.Procfile, error)

// Options is provided to New to configure the Empire services.
type Options struct {
	Secret string
}

// Empire is a context object that contains a collection of services.
type Empire struct {
	// Reporter is an reporter.Reporter that will be used to report errors to
	// an external system.
	reporter.Reporter

	// Logger is a log15 logger that will be used for logging.
	Logger log15.Logger

	store *store

	accessTokens *accessTokensService
	apps         *appsService
	configs      *configsService
	domains      *domainsService
	tasks        *tasksService
	releases     *releasesService
	releaser     *releaser
	deployer     *deployerService
	scaler       *scaler
	restarter    *restarter
	runner       *runnerService
	slugs        *slugsService
	certs        *certsService

	// Scheduler is the backend scheduler used to run applications.
	Scheduler scheduler.Scheduler

	// LogsStreamer is the backend used to stream application logs.
	LogsStreamer LogsStreamer

	// ExtractProcfile is called during deployments to extract the Procfile
	// from the newly deployed image.
	ExtractProcfile ProcfileExtractor

	// EventStream service for publishing Empire events.
	EventStream EventStream
}

// New returns a new Empire instance.
func New(db *gorm.DB, options Options) *Empire {
	e := &Empire{
		Logger:       nullLogger(),
		LogsStreamer: logsDisabled,
		EventStream:  NullEventStream,
		store:        &store{db: db},
	}

	e.accessTokens = &accessTokensService{Secret: []byte(options.Secret)}
	e.apps = &appsService{Empire: e}
	e.configs = &configsService{Empire: e}
	e.deployer = &deployerService{Empire: e}
	e.domains = &domainsService{Empire: e}
	e.slugs = &slugsService{Empire: e}
	e.tasks = &tasksService{Empire: e}
	e.scaler = &scaler{Empire: e}
	e.restarter = &restarter{Empire: e}
	e.runner = &runnerService{Empire: e}
	e.releases = &releasesService{Empire: e}
	e.releaser = &releaser{Empire: e}
	e.certs = &certsService{Empire: e}
	return e
}

// AccessTokensFind finds an access token.
func (e *Empire) AccessTokensFind(token string) (*AccessToken, error) {
	return e.accessTokens.AccessTokensFind(token)
}

// AccessTokensCreate creates a new AccessToken.
func (e *Empire) AccessTokensCreate(accessToken *AccessToken) (*AccessToken, error) {
	return e.accessTokens.AccessTokensCreate(accessToken)
}

// AppsFirst finds the first app matching the query.
func (e *Empire) AppsFirst(q AppsQuery) (*App, error) {
	return e.store.AppsFirst(q)
}

// Apps returns all Apps.
func (e *Empire) Apps(q AppsQuery) ([]*App, error) {
	return e.store.Apps(q)
}

// AppsCreate creates a new app.
func (e *Empire) AppsCreate(app *App) (*App, error) {
	return e.store.AppsCreate(app)
}

// AppsDestroy destroys the app.
func (e *Empire) AppsDestroy(ctx context.Context, app *App) error {
	return e.apps.AppsDestroy(ctx, app)
}

// Config returns the current Config for a given app.
func (e *Empire) Config(app *App) (*Config, error) {
	return e.configs.Config(app)
}

// ConfigsApply applies the new config vars to the apps current Config,
// returning a new Config. If the app has a running release, a new release will
// be created and run.
func (e *Empire) ConfigsApply(ctx context.Context, app *App, vars Vars) (*Config, error) {
	return e.configs.ConfigsApply(ctx, app, vars)
}

// DomainsFirst returns the first domain matching the query.
func (e *Empire) DomainsFirst(q DomainsQuery) (*Domain, error) {
	return e.store.DomainsFirst(q)
}

// Domains returns all domains matching the query.
func (e *Empire) Domains(q DomainsQuery) ([]*Domain, error) {
	return e.store.Domains(q)
}

// DomainsCreate adds a new Domain for an App.
func (e *Empire) DomainsCreate(domain *Domain) (*Domain, error) {
	return e.domains.DomainsCreate(domain)
}

// DomainsDestroy removes a Domain for an App.
func (e *Empire) DomainsDestroy(domain *Domain) error {
	return e.domains.DomainsDestroy(domain)
}

// Tasks returns the Tasks for the given app.
func (e *Empire) Tasks(ctx context.Context, app *App) ([]*Task, error) {
	return e.tasks.Tasks(ctx, app)
}

// RestartOpts are options provided when restarting an app.
type RestartOpts struct {
	// User performing the action.
	User *User

	// The associated app.
	App *App

	// If provided, a PID that will be killed. Generally used for killing
	// detached processes.
	PID string
}

func (opts RestartOpts) Event() Event {
	return RestartEvent{
		User: opts.User.Name,
		App:  opts.App.Name,
		PID:  opts.PID,
	}
}

// Restart restarts processes matching the given prefix for the given Release.
// If the prefix is empty, it will match all processes for the release.
func (e *Empire) Restart(ctx context.Context, opts RestartOpts) error {
	if err := e.restarter.Restart(ctx, opts); err != nil {
		return err
	}

	return e.EventStream.PublishEvent(opts.Event())

}

// RunOpts are options provided when running an attached/detached process.
type RunOpts struct {
	// User performing this action.
	User *User

	// Related app.
	App *App

	// The command to run.
	Command string

	// If provided, input will be read from this.
	Input io.Reader

	// If provided, output will be written to this.
	Output io.Writer

	// Extra environment variables to set.
	Env map[string]string
}

func (opts RunOpts) Event() Event {
	var attached bool
	if opts.Output != nil {
		attached = true
	}

	return RunEvent{
		User:     opts.User.Name,
		App:      opts.App.Name,
		Command:  opts.Command,
		Attached: attached,
	}
}

// Run runs a one-off process for a given App and command.
func (e *Empire) Run(ctx context.Context, opts RunOpts) error {
	if err := e.runner.Run(ctx, opts); err != nil {
		return err
	}

	return e.EventStream.PublishEvent(opts.Event())
}

// Releases returns all Releases for a given App.
func (e *Empire) Releases(q ReleasesQuery) ([]*Release, error) {
	return e.store.Releases(q)
}

// ReleasesFirst returns the first releases for a given App.
func (e *Empire) ReleasesFirst(q ReleasesQuery) (*Release, error) {
	return e.store.ReleasesFirst(q)
}

// ReleasesLast returns the last release for an App.
func (e *Empire) ReleasesLast(app *App) (*Release, error) {
	return e.store.ReleasesFirst(ReleasesQuery{App: app})
}

// RollbackOpts are options provided when rolling back to an old release.
type RollbackOpts struct {
	// The user performing the action.
	User *User

	// The associated app.
	App *App

	// The release version to rollback to.
	Version int
}

func (opts RollbackOpts) Event() Event {
	return RollbackEvent{
		User:    opts.User.Name,
		App:     opts.App.Name,
		Version: opts.Version,
	}
}

// Rollback rolls an app back to a specific release version. Returns a
// new release.
func (e *Empire) Rollback(ctx context.Context, opts RollbackOpts) (*Release, error) {
	r, err := e.releases.Rollback(ctx, opts)
	if err != nil {
		return r, err
	}

	return r, e.EventStream.PublishEvent(opts.Event())
}

// DeploymentsCreateOpts represents options that can be passed when deploying to
// an application.
type DeploymentsCreateOpts struct {
	// User the user that is triggering the deployment.
	User *User

	// App is the app that is being deployed to.
	App *App

	// Image is the image that's being deployed.
	Image image.Image

	// Output is an io.Writer where deployment output and events will be
	// streamed in jsonmessage format.
	Output io.Writer
}

func (opts DeploymentsCreateOpts) Event() Event {
	e := DeployEvent{
		User:  opts.User.Name,
		Image: opts.Image.String(),
	}
	if opts.App != nil {
		e.App = opts.App.Name
	}
	return e
}

// Deploy deploys an image and streams the output to w.
func (e *Empire) Deploy(ctx context.Context, opts DeploymentsCreateOpts) (*Release, error) {
	r, err := e.deployer.Deploy(ctx, opts)
	if err != nil {
		return r, err
	}

	return r, e.EventStream.PublishEvent(opts.Event())
}

// ScaleOpts are options provided when scaling a process.
type ScaleOpts struct {
	// User that's performing the action.
	User *User

	// The associated app.
	App *App

	// The process type to scale.
	Process ProcessType

	// The desired quantity of processes.
	Quantity int

	// If provided, new memory and CPU constraints for the process.
	Constraints *Constraints
}

func (opts ScaleOpts) Event() Event {
	return ScaleEvent{
		User:     opts.User.Name,
		App:      opts.App.Name,
		Process:  string(opts.Process),
		Quantity: opts.Quantity,
	}
}

// Scale scales an apps process.
func (e *Empire) Scale(ctx context.Context, opts ScaleOpts) (*Process, error) {
	p, err := e.scaler.Scale(ctx, opts)
	if err != nil {
		return p, err
	}

	return p, e.EventStream.PublishEvent(opts.Event())
}

// Streamlogs streams logs from an app.
func (e *Empire) StreamLogs(app *App, w io.Writer) error {
	return e.LogsStreamer.StreamLogs(app, w)
}

// CertsAttach attaches an SSL certificate to the app.
func (e *Empire) CertsAttach(ctx context.Context, app *App, cert string) error {
	return e.certs.CertsAttach(ctx, app, cert)
}

// Reset resets empire.
func (e *Empire) Reset() error {
	return e.store.Reset()
}

// IsHealthy returns true if Empire is healthy, which means it can connect to
// the services it depends on.
func (e *Empire) IsHealthy() bool {
	return e.store.IsHealthy()
}

// Migrate runs the migrations.
func Migrate(db, path string) ([]error, bool) {
	return migrate.UpSync(db, path)
}

// ValidationError is returned when a model is not valid.
type ValidationError struct {
	Err error
}

func (e *ValidationError) Error() string {
	return e.Err.Error()
}

func newJSONMessageError(err error) jsonmessage.JSONMessage {
	return jsonmessage.JSONMessage{
		ErrorMessage: err.Error(),
		Error: &jsonmessage.JSONError{
			Message: err.Error(),
		},
	}
}

func nullLogger() log15.Logger {
	l := log15.New()
	h := log15.StreamHandler(ioutil.Discard, log15.LogfmtFormat())
	l.SetHandler(h)
	return l
}

// PullAndExtract returns a ProcfileExtractor that will pull the image using the
// docker client, then attempt to extract the Procfile from the WORKDIR, or
// fallback to the CMD directive in the Procfile.
func PullAndExtract(c *dockerutil.Client) ProcfileExtractor {
	e := procfile.MultiExtractor(
		procfile.NewFileExtractor(c.Client),
		procfile.NewCMDExtractor(c.Client),
	)

	return ProcfileExtractor(func(ctx context.Context, img image.Image, w io.Writer) (procfile.Procfile, error) {
		repo := img.Repository
		if img.Registry != "" {
			repo = fmt.Sprintf("%s/%s", img.Registry, img.Repository)
		}

		if err := c.PullImage(ctx, docker.PullImageOptions{
			Registry:      img.Registry,
			Repository:    repo,
			Tag:           img.Tag,
			OutputStream:  w,
			RawJSONStream: true,
		}); err != nil {
			return nil, err
		}

		return e.Extract(img)
	})
}
