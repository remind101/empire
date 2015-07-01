package empire // import "github.com/remind101/empire/empire"

import (
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/fsouza/go-dockerclient"
	"github.com/inconshreveable/log15"
	"github.com/mattes/migrate/migrate"
	"github.com/remind101/empire/empire/pkg/dockerutil"
	"github.com/remind101/empire/empire/pkg/image"
	"github.com/remind101/empire/empire/pkg/runner"
	"github.com/remind101/empire/empire/pkg/service"
	"github.com/remind101/empire/empire/pkg/sslcert"
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

// DockerOptions is a set of options to configure a docker api client.
type DockerOptions struct {
	// The unix socket to connect to the docker api.
	Socket string

	// Path to a certificate to use for TLS connections.
	CertPath string

	// A set of docker registry credentials.
	Auth *docker.AuthConfigurations
}

// ECSOptions is a set of options to configure ECS.
type ECSOptions struct {
	Cluster     string
	ServiceRole string
}

// ELBOptions is a set of options to configure ELB.
type ELBOptions struct {
	// The Security Group ID to assign when creating internal load balancers.
	InternalSecurityGroupID string

	// The Security Group ID to assign when creating external load balancers.
	ExternalSecurityGroupID string

	// The Subnet IDs to assign when creating internal load balancers.
	InternalSubnetIDs []string

	// The Subnet IDs to assign when creating external load balancers.
	ExternalSubnetIDs []string

	// Zone ID of the internal zone to add cnames for each elb
	InternalZoneID string
}

// Options is provided to New to configure the Empire services.
type Options struct {
	Docker DockerOptions
	ECS    ECSOptions
	ELB    ELBOptions

	// AWS Configuration
	AWSConfig *aws.Config

	Secret string

	// Database connection string.
	DB string
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
	deployer     *deployer
	scaler       *scaler
	restarter    *restarter
	runner       *runnerService
}

// New returns a new Empire instance.
func New(options Options) (*Empire, error) {
	db, err := newDB(options.DB)
	if err != nil {
		return nil, err
	}

	store := &store{db: db}

	extractor, err := newExtractor(options.Docker)
	if err != nil {
		return nil, err
	}

	resolver, err := newResolver(options.Docker)
	if err != nil {
		return nil, err
	}

	runner, err := newRunner(options.Docker)
	if err != nil {
		return nil, err
	}

	manager, err := newManager(
		runner,
		options.ECS,
		options.ELB,
		options.AWSConfig,
	)
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

	releaser := &releaser{
		store:   store,
		manager: manager,
	}

	restarter := &restarter{
		releaser: releaser,
		manager:  manager,
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
		store: store,
	}

	slugs := &slugsService{
		store:     store,
		extractor: extractor,
		resolver:  resolver,
	}

	deployer := &deployer{
		appsService:     apps,
		configsService:  configs,
		slugsService:    slugs,
		releasesService: releases,
	}

	certs := &certificatesService{
		store:    store,
		manager:  newCertManager(options.AWSConfig),
		releaser: releaser,
	}

	return &Empire{
		Logger:       newLogger(),
		store:        store,
		accessTokens: accessTokens,
		apps:         apps,
		certs:        certs,
		configs:      configs,
		deployer:     deployer,
		domains:      domains,
		jobStates:    jobStates,
		scaler:       scaler,
		restarter:    restarter,
		runner: &runnerService{
			store:   store,
			manager: manager,
		},
		releases: releases,
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

// ReleasesFindByApp returns all Releases for a given App.
func (e *Empire) ReleasesFindByApp(app *App) ([]*Release, error) {
	return e.store.Releases(ReleasesQuery{App: app})
}

// ReleasesFindByAppAndVersion finds a specific Release for a given App.
func (e *Empire) ReleasesFindByAppAndVersion(app *App, version int) (*Release, error) {
	return e.store.ReleasesFirst(ReleasesQuery{App: app, Version: &version})
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

// DeployImage deploys an image to Empire.
func (e *Empire) DeployImage(ctx context.Context, img image.Image, out chan Event) (*Release, error) {
	return e.deployer.DeployImage(ctx, img, out)
}

// AppsScale scales an apps process.
func (e *Empire) AppsScale(ctx context.Context, app *App, t ProcessType, quantity int, c *Constraints) (*Process, error) {
	return e.scaler.Scale(ctx, app, t, quantity, c)
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

func newManager(r *runner.Runner, ecsOpts ECSOptions, elbOpts ELBOptions, config *aws.Config) (service.Manager, error) {
	if config == nil {
		log.Println("warn: AWS not configured, ECS service management disabled.")
		return service.NewFakeManager(), nil
	}

	m, err := service.NewLoadBalancedECSManager(service.ECSConfig{
		Cluster:                 ecsOpts.Cluster,
		ServiceRole:             ecsOpts.ServiceRole,
		InternalSecurityGroupID: elbOpts.InternalSecurityGroupID,
		ExternalSecurityGroupID: elbOpts.ExternalSecurityGroupID,
		InternalSubnetIDs:       elbOpts.InternalSubnetIDs,
		ExternalSubnetIDs:       elbOpts.ExternalSubnetIDs,
		AWS:                     config,
		ZoneID:                  elbOpts.InternalZoneID,
	})
	if err != nil {
		return nil, err
	}

	return &service.AttachedRunner{
		Manager: m,
		Runner:  r,
	}, nil
}

func newCertManager(config *aws.Config) sslcert.Manager {
	if config == nil {
		log.Println("warn: AWS not configured, IAM server certificate management disabled.")
		return sslcert.NewFakeManager()
	}

	return sslcert.NewIAMManager(config, "/empire/certs/")
}

func newRunner(o DockerOptions) (*runner.Runner, error) {
	if o.Socket == "" {
		return nil, nil
	}

	c, err := dockerutil.NewClient(o.Auth, o.Socket, o.CertPath)
	if err != nil {
		return nil, err
	}

	return runner.NewRunner(c), nil
}

func newLogger() log15.Logger {
	l := log15.New()
	h := log15.StreamHandler(os.Stdout, log15.LogfmtFormat())
	//h = log15.CallerStackHandler("%+n", h)
	l.SetHandler(log15.LazyHandler(h))
	return l
}

func newExtractor(o DockerOptions) (Extractor, error) {
	if o.Socket == "" {
		log.Println("warn: docker socket not configured, docker command extractor disabled.")
		return &fakeExtractor{}, nil
	}

	c, err := dockerutil.NewDockerClient(o.Socket, o.CertPath)
	return newProcfileFallbackExtractor(c), err
}

func newResolver(o DockerOptions) (Resolver, error) {
	if o.Socket == "" {
		log.Println("warn: docker socket not configured, docker image puller disabled.")
		return &fakeResolver{}, nil
	}

	c, err := dockerutil.NewClient(o.Auth, o.Socket, o.CertPath)
	return newDockerResolver(c), err
}
