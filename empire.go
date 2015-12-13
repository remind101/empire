package empire // import "github.com/remind101/empire"

import (
	"io"
	"io/ioutil"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/fsouza/go-dockerclient"
	"github.com/inconshreveable/log15"
	"github.com/jinzhu/gorm"
	"github.com/mattes/migrate/migrate"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/pkg/sslcert"
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
	certs        *certificatesService
	configs      *configsService
	domains      *domainsService
	jobStates    *processStatesService
	releases     *releasesService
	releaser     *releaser
	deployer     *deployerService
	scaler       *scaler
	restarter    *restarter
	runner       *runnerService
	slugs        *slugsService

	// Scheduler is the backend scheduler used to run applications.
	Scheduler scheduler.Scheduler

	// CertManager is the backend used to store SSL/TLS certificates.
	CertManager sslcert.Manager

	// LogsStreamer is the backend used to stream application logs.
	LogsStreamer LogsStreamer

	// ExtractProcfile is called during deployments to extract the Procfile
	// from the newly deployed image.
	ExtractProcfile ProcfileExtractor
}

// New returns a new Empire instance.
func New(db *gorm.DB, options Options) *Empire {
	e := &Empire{
		Logger:       nullLogger(),
		LogsStreamer: logsDisabled,
		store:        &store{db: db},
	}

	e.accessTokens = &accessTokensService{Secret: []byte(options.Secret)}
	e.apps = &appsService{Empire: e}
	e.certs = &certificatesService{Empire: e}
	e.configs = &configsService{Empire: e}
	e.deployer = &deployerService{Empire: e}
	e.domains = &domainsService{Empire: e}
	e.slugs = &slugsService{Empire: e}
	e.jobStates = &processStatesService{Empire: e}
	e.scaler = &scaler{Empire: e}
	e.restarter = &restarter{Empire: e}
	e.runner = &runnerService{Empire: e}
	e.releases = &releasesService{Empire: e}
	e.releaser = &releaser{Empire: e}
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

// CertificatesFirst returns a certificate for the given ID
func (e *Empire) CertificatesFirst(ctx context.Context, q CertificatesQuery) (*Certificate, error) {
	return e.store.CertificatesFirst(q)
}

// CertificatesCreate creates a certificate.
func (e *Empire) CertificatesCreate(ctx context.Context, cert *Certificate) (*Certificate, error) {
	return e.certs.CertificatesCreate(ctx, cert)
}

// CertificatesUpdate updates a certificate.
func (e *Empire) CertificatesUpdate(ctx context.Context, cert *Certificate) (*Certificate, error) {
	return e.certs.CertificatesUpdate(ctx, cert)
}

// CertificatesDestroy destroys a certificate.
func (e *Empire) CertificatesDestroy(ctx context.Context, cert *Certificate) error {
	return e.certs.CertificatesDestroy(ctx, cert)
}

// ConfigsCurrent returns the current Config for a given app.
func (e *Empire) ConfigsCurrent(app *App) (*Config, error) {
	return e.configs.ConfigsCurrent(app)
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

// JobStatesByApp returns the JobStates for the given app.
func (e *Empire) JobStatesByApp(ctx context.Context, app *App) ([]*ProcessState, error) {
	return e.jobStates.JobStatesByApp(ctx, app)
}

// ProcessesRestart restarts processes matching the given prefix for the given Release.
// If the prefix is empty, it will match all processes for the release.
func (e *Empire) ProcessesRestart(ctx context.Context, app *App, id string) error {
	return e.restarter.Restart(ctx, app, id)
}

// ProcessesRun runs a one-off process for a given App and command.
func (e *Empire) ProcessesRun(ctx context.Context, app *App, opts ProcessRunOpts) error {
	return e.runner.Run(ctx, app, opts)
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

// ReleasesRollback rolls an app back to a specific release version. Returns a
// new release.
func (e *Empire) ReleasesRollback(ctx context.Context, app *App, version int) (*Release, error) {
	return e.releases.ReleasesRollback(ctx, app, version)
}

// Deploy deploys an image and streams the output to w.
func (e *Empire) Deploy(ctx context.Context, opts DeploymentsCreateOpts) (*Release, error) {
	return e.deployer.Deploy(ctx, opts)
}

// AppsScale scales an apps process.
func (e *Empire) AppsScale(ctx context.Context, app *App, t ProcessType, quantity int, c *Constraints) (*Process, error) {
	return e.scaler.Scale(ctx, app, t, quantity, c)
}

// Streamlogs streams logs from an app.
func (e *Empire) StreamLogs(app *App, w io.Writer) error {
	return e.LogsStreamer.StreamLogs(app, w)
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

// key used to store context values from within this package.
type key int

const (
	UserKey key = 0
)

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
		if err := c.PullImage(ctx, docker.PullImageOptions{
			Registry:      img.Registry,
			Repository:    img.Repository,
			Tag:           img.Tag,
			OutputStream:  w,
			RawJSONStream: true,
		}); err != nil {
			return nil, err
		}

		return e.Extract(img)
	})
}
