package empire // import "github.com/remind101/empire/empire"

import (
	"net/url"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/mattes/migrate/migrate"
	"github.com/remind101/empire/empire/pkg/container"
)

// A function to return the current time. It can be useful to stub this out in
// tests.
var Now = func() time.Time {
	return time.Now().UTC()
}

// DefaultOptions is a default Options instance that can be passed when
// intializing a new Empire.
var DefaultOptions = Options{}

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
	AccessTokens *accessTokensService
	Apps         *appsService
	Configs      *configsService
	JobStates    *jobStatesService
	Manager      *manager
	Releases     *releasesService
	Slugs        *slugsService
	Deployer     *deployer

	store *store
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

	configs := &configsService{
		store: store,
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

	slugs := &slugsService{
		store:     store,
		extractor: extractor,
	}

	deployer := &deployer{
		Organization:    options.Docker.Organization,
		appsService:     apps,
		configsService:  configs,
		slugsService:    slugs,
		releasesService: releases,
	}

	return &Empire{
		store:        store,
		AccessTokens: accessTokens,
		Apps:         apps,
		Configs:      configs,
		Deployer:     deployer,
		JobStates:    jobStates,
		Manager:      manager,
		Slugs:        slugs,
		Releases:     releases,
	}, nil
}

// AccessTokensFind finds an access token.
func (e *Empire) AccessTokensFind(token string) (*AccessToken, error) {
	return e.AccessTokens.AccessTokensFind(token)
}

// AccessTokensCreate creates a new AccessToken.
func (e *Empire) AccessTokensCreate(accessToken *AccessToken) (*AccessToken, error) {
	return e.AccessTokens.AccessTokensCreate(accessToken)
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
func (e *Empire) AppsDestroy(app *App) error {
	return e.Apps.AppsDestroy(app)
}

// ConfigsCurrent returns the current Config for a given app.
func (e *Empire) ConfigsCurrent(app *App) (*Config, error) {
	return e.Configs.ConfigsCurrent(app)
}

// ConfigsApply applies the new config vars to the apps current Config,
// returning a new Config.
func (e *Empire) ConfigsApply(app *App, vars Vars) (*Config, error) {
	return e.Configs.ConfigsApply(app, vars)
}

// ConfigsFind finds a Config by id.
func (e *Empire) ConfigsFind(id string) (*Config, error) {
	return e.store.ConfigsFind(id)
}

// JobStatesByApp returns the JobStates for the given app.
func (e *Empire) JobStatesByApp(app *App) ([]*JobState, error) {
	return e.JobStates.JobStatesByApp(app)
}

// ProcessesAll returns all processes for a given Release.
func (e *Empire) ProcessesAll(release *Release) (Formation, error) {
	return e.store.ProcessesAll(release)
}

// ReleasesCreate creates a new release for an app.
func (e *Empire) ReleasesCreate(app *App, config *Config, slug *Slug, desc string) (*Release, error) {
	return e.Releases.ReleasesCreate(app, config, slug, desc)
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

// ScaleRelease scales the processes in a release.
func (e *Empire) ScaleRelease(release *Release, config *Config, slug *Slug, formation Formation, qm ProcessQuantityMap) error {
	return e.Manager.ScaleRelease(release, config, slug, formation, qm)
}

// SlugsFind finds a slug by id.
func (e *Empire) SlugsFind(id string) (*Slug, error) {
	return e.store.SlugsFind(id)
}

// DeployImage deploys an image to Empire.
func (e *Empire) DeployImage(image Image) (*Deploy, error) {
	return e.Deployer.DeployImage(image)
}

// DeployCommit deploys a Commit to Empire.
func (e *Empire) DeployCommit(commit Commit) (*Deploy, error) {
	return e.Deployer.DeployCommit(commit)
}

// Reset resets empire.
func (e *Empire) Reset() error {
	return e.store.Reset()
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
