package empire // import "github.com/remind101/empire/empire"

import (
	"net/url"
	"strings"

	"github.com/fsouza/go-dockerclient"
	"github.com/mattes/migrate/migrate"
	"github.com/remind101/empire/empire/pkg/container"
	"github.com/remind101/empire/empire/pkg/pod"
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

// EtcdOptions is a set of options to configure an etcd api client.
type EtcdOptions struct {
	// The etcd hosts to connect to.
	API string
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
	Etcd   EtcdOptions

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
	domains      *domainsService
	jobStates    *processStatesService
	releases     *releasesService
	deployer     *deployer
	scaler       *scaler
	restarter    *restarter
	releaser     *releaser
}

// New returns a new Empire instance.
func New(options Options) (*Empire, error) {
	db, err := newDB(options.DB)
	if err != nil {
		return nil, err
	}

	store := &store{db: db}

	domainReg := newDomainRegistry(options.Etcd.API)

	extractor, err := NewExtractor(
		options.Docker.Socket,
		options.Docker.CertPath,
	)
	if err != nil {
		return nil, err
	}

	resolver, err := newResolver(
		options.Docker.Socket,
		options.Docker.CertPath,
		options.Docker.Auth,
	)
	if err != nil {
		return nil, err
	}

	manager, err := newManager(options)
	if err != nil {
		return nil, err
	}

	accessTokens := &accessTokensService{
		Secret: []byte(options.Secret),
	}

	apps := &appsService{
		store:   store,
		manager: manager,
	}

	jobStates := &processStatesService{
		manager: manager,
	}

	scaler := &scaler{
		store:   store,
		manager: manager,
	}

	restarter := &restarter{
		manager: manager,
	}

	releaser := &releaser{
		manager: manager,
	}

	releases := &releasesService{
		store:    store,
		releaser: releaser,
	}

	configs := &configsService{
		store:    store,
		releases: releases,
	}

	domains := &domainsService{
		store:    store,
		registry: domainReg,
	}

	slugs := &slugsService{
		store:     store,
		extractor: extractor,
		resolver:  resolver,
	}

	deployer := &deployer{
		Organization:    options.Docker.Organization,
		store:           store,
		appsService:     apps,
		configsService:  configs,
		slugsService:    slugs,
		releasesService: releases,
	}

	return &Empire{
		store:        store,
		accessTokens: accessTokens,
		apps:         apps,
		configs:      configs,
		deployer:     deployer,
		domains:      domains,
		jobStates:    jobStates,
		releaser:     releaser,
		scaler:       scaler,
		restarter:    restarter,
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

// DomainsByApp returns the domains for a given App.
func (e *Empire) DomainsFindByApp(app *App) ([]*Domain, error) {
	return e.store.DomainsFindByApp(app)
}

// DomainsByHostname returns the domain for a given hostname.
func (e *Empire) DomainsFindByHostname(hostname string) (*Domain, error) {
	return e.store.DomainsFindByHostname(hostname)
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
func (e *Empire) JobStatesByApp(app *App) ([]*ProcessState, error) {
	return e.jobStates.JobStatesByApp(app)
}

// ProcessesAll returns all processes for a given Release.
func (e *Empire) ProcessesAll(release *Release) (Formation, error) {
	return e.store.ProcessesAll(release)
}

// ProcessesRestart restarts processes matching the given prefix for the given Release.
// If the prefix is empty, it will match all processes for the release.
func (e *Empire) ProcessesRestart(ctx context.Context, app *App, ptype ProcessType, pnum int) error {
	return e.restarter.Restart(ctx, app, ptype, pnum)
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

func newManager(options Options) (*manager, error) {
	scheduler, err := newScheduler(options.Fleet.API)
	if err != nil {
		return nil, err
	}

	var store pod.Store
	switch options.Etcd.API {
	case "fake":
		store = pod.NewMemStore()
	default:
		machines := strings.Split(options.Etcd.API, ",")
		store = pod.NewEtcdStore(machines)
	}

	return &manager{
		Manager: pod.NewContainerManager(scheduler, store),
	}, nil
}
