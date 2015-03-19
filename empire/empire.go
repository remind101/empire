package empire // import "github.com/remind101/empire/empire"

import (
	"net/url"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/mattes/migrate/migrate"
	"github.com/remind101/empire/empire/pkg/container"
	"github.com/remind101/empire/empire/pkg/reporter"
	"golang.org/x/net/context"
)

// A function to return the current time. It can be useful to stub this out in
// tests.
var Now = func() time.Time {
	return time.Now().UTC()
}

var (
	// DefaultOptions is a default Options instance that can be passed when
	// intializing a new Empire.
	DefaultOptions = Options{}

	// defaultReporter is the default reporter.Reporter to use.
	defaultReporter = reporter.NewLogReporter()
)

// DockerOptions is a set of options to configure a docker api client.
type DockerOptions struct {
	// The default docker organization to use.
	Organization string

	// The unix socket to connect to the docker api.
	Socket string

	// Path to a certificate to use for TLS connections.
	CertPath string

	// A set of docker registry credentials.
	Auth *docker.AuthConfigurations
}

// FleetOptions is a set of options to configure a fleet api client.
type FleetOptions struct {
	// The location of the fleet api.
	API string
}

// Options is provided to New to configure the Empire services.
type Options struct {
	Docker DockerOptions
	Fleet  FleetOptions

	Secret string

	// Database connection string.
	DB string
}

// Empire is a context object that contains a collection of services.
type Empire struct {
	// Reporter is an reporter.Reporter that will be used to report errors to
	// an external system.
	reporter.Reporter

	store *store

	accessTokens *accessTokensService
	apps         *appsService
	configs      *configsService
	jobStates    *jobStatesService
	manager      *manager
	releases     *releasesService
	deployer     *deployer
	scaler       *scaler
}

// New returns a new Empire instance.
func New(options Options) (*Empire, error) {
	db, err := newDB(options.DB)
	if err != nil {
		return nil, err
	}

	store := &store{db: db}

	scheduler, err := newScheduler(options.Fleet.API)
	if err != nil {
		return nil, err
	}

	extractor, err := NewExtractor(
		options.Docker.Socket,
		options.Docker.CertPath,
		options.Docker.Auth,
	)
	if err != nil {
		return nil, err
	}

	accessTokens := &accessTokensService{
		Secret: []byte(options.Secret),
	}

	jobs := &jobsService{
		store:     store,
		scheduler: scheduler,
	}

	jobStates := &jobStatesService{
		store:     store,
		scheduler: scheduler,
	}

	apps := &appsService{
		store:       store,
		jobsService: jobs,
	}

	manager := &manager{
		jobsService: jobs,
		store:       store,
	}

	releases := &releasesService{
		store:   store,
		manager: manager,
	}

	configs := &configsService{
		store:    store,
		releases: releases,
	}

	slugs := &slugsService{
		store:     store,
		extractor: extractor,
	}

	deployer := &deployer{
		Organization:    options.Docker.Organization,
		store:           store,
		appsService:     apps,
		configsService:  configs,
		slugsService:    slugs,
		releasesService: releases,
	}

	scaler := &scaler{
		store:   store,
		manager: manager,
	}

	return &Empire{
		Reporter:     defaultReporter,
		store:        store,
		accessTokens: accessTokens,
		apps:         apps,
		configs:      configs,
		deployer:     deployer,
		jobStates:    jobStates,
		manager:      manager,
		scaler:       scaler,
		releases:     releases,
	}, nil
}

// AccessTokensFind finds an access token.
func (e *Empire) AccessTokensFind(token string) (*AccessToken, error) {
	return e.accessTokens.AccessTokensFind(token)
}

// AccessTokensCreate creates a new AccessToken.
func (e *Empire) AccessTokensCreate(accessToken *AccessToken) (*AccessToken, error) {
	return e.accessTokens.AccessTokensCreate(accessToken)
}

// AppsAll returns all Apps.
func (e *Empire) AppsAll() ([]*App, error) {
	return e.store.AppsAll()
}

// AppsCreate creates a new app.
func (e *Empire) AppsCreate(app *App) (*App, error) {
	return e.store.AppsCreate(app)
}

// AppsFind finds an app by name.
func (e *Empire) AppsFind(name string) (*App, error) {
	return e.store.AppsFind(name)
}

// AppsDestroy destroys the app.
func (e *Empire) AppsDestroy(ctx context.Context, app *App) error {
	return e.apps.AppsDestroy(ctx, app)
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

// JobStatesByApp returns the JobStates for the given app.
func (e *Empire) JobStatesByApp(app *App) ([]*JobState, error) {
	return e.jobStates.JobStatesByApp(app)
}

// ProcessesAll returns all processes for a given Release.
func (e *Empire) ProcessesAll(release *Release) (Formation, error) {
	return e.store.ProcessesAll(release)
}

// ReleasesFindByApp returns all Releases for a given App.
func (e *Empire) ReleasesFindByApp(app *App) ([]*Release, error) {
	return e.store.ReleasesFindByApp(app)
}

// ReleasesFindByAppAndVersion finds a specific Release for a given App.
func (e *Empire) ReleasesFindByAppAndVersion(app *App, version int) (*Release, error) {
	return e.store.ReleasesFindByAppAndVersion(app, version)
}

// ReleasesLast returns the last release for an App.
func (e *Empire) ReleasesLast(app *App) (*Release, error) {
	return e.store.ReleasesLast(app)
}

// ReleasesRollback rolls an app back to a specific release version. Returns a
// new release.
func (e *Empire) ReleasesRollback(ctx context.Context, app *App, version int) (*Release, error) {
	return e.releases.ReleasesRollback(ctx, app, version)
}

// DeployImage deploys an image to Empire.
func (e *Empire) DeployImage(ctx context.Context, image Image) (*Deployment, error) {
	return e.deployer.DeployImage(ctx, image)
}

// DeployCommit deploys a Commit to Empire.
func (e *Empire) DeployCommit(ctx context.Context, commit Commit) (*Deployment, error) {
	return e.deployer.DeployCommit(ctx, commit)
}

// AppsScale scales an apps process.
func (e *Empire) AppsScale(ctx context.Context, app *App, t ProcessType, quantity int) error {
	return e.scaler.Scale(ctx, app, t, quantity)
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

func newScheduler(fleetURL string) (container.Scheduler, error) {
	if fleetURL == "fake" {
		return container.NewFakeScheduler(), nil
	}

	u, err := url.Parse(fleetURL)
	if err != nil {
		return nil, err
	}

	return container.NewFleetScheduler(u)
}
