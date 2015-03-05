package empire // import "github.com/remind101/empire/empire"

import (
	"time"

	"github.com/mattes/migrate/migrate"
	"github.com/remind101/empire/empire/scheduler"
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
	// The unix socket to connect to the docker api.
	Socket string

	// Path to a certificate to use for TLS connections.
	CertPath string

	// Path to a docker registry auth file. Typically ~/.dockercfg
	AuthPath string
}

// FleetOptions is a set of options to configure a fleet api client.
type FleetOptions struct {
	// The location of the fleet api.
	API string
}

// GitHub options is a set of options to configure the GitHub integration.
type GitHubOptions struct {
	// Secret is the shared secret for GitHub webhooks.
	Secret string

	// Token is the GitHub OAuth token to use when creating
	// DeploymentStatuses.
	Token string
}

// Options is provided to New to configure the Empire services.
type Options struct {
	Docker DockerOptions
	Fleet  FleetOptions
	GitHub GitHubOptions

	// Database connection string.
	DB string
}

// Empire is a context object that contains a collection of services.
type Empire struct {
	// Stored options
	Options *Options

	DB DB

	AppsService
	ConfigsService
	JobsService
	JobStatesService
	Manager
	ReleasesService
	SlugsService
	ProcessesService

	*DeploysService
	*GitHubDeploysService
}

// New returns a new Empire instance.
func New(options Options) (*Empire, error) {
	db, err := NewDB(options.DB)
	if err != nil {
		return nil, err
	}

	scheduler, err := scheduler.NewScheduler(options.Fleet.API)
	if err != nil {
		return nil, err
	}

	extractor, err := NewExtractor(
		options.Docker.Socket,
		options.Docker.CertPath,
		options.Docker.AuthPath,
	)
	if err != nil {
		return nil, err
	}

	apps := &appsService{
		DB: db,
	}

	configs := &configsService{
		DB: db,
	}

	jobs := &jobsService{
		DB:        db,
		scheduler: scheduler,
	}

	jobStates := &jobStatesService{
		DB:          db,
		JobsService: jobs,
		scheduler:   scheduler,
	}

	processes := &processesService{
		DB: db,
	}

	manager := &manager{
		JobsService:      jobs,
		ProcessesService: processes,
	}

	releases := &releasesService{
		DB:               db,
		ProcessesService: processes,
		Manager:          manager,
	}

	slugs := &slugsService{
		DB:        db,
		extractor: extractor,
	}

	deploys := &DeploysService{
		AppsService:     apps,
		ConfigsService:  configs,
		SlugsService:    slugs,
		ReleasesService: releases,
	}

	ghDeploys := &GitHubDeploysService{
		DeploysService: deploys,
		resolver: &RegistryResolver{
			Registry: "quay.io",
		},
	}

	return &Empire{
		Options:          &options,
		DB:               db,
		AppsService:      apps,
		ConfigsService:   configs,
		JobsService:      jobs,
		JobStatesService: jobStates,
		Manager:          manager,
		SlugsService:     slugs,
		ReleasesService:  releases,
		ProcessesService: processes,

		DeploysService:       deploys,
		GitHubDeploysService: ghDeploys,
	}, nil
}

func (e *Empire) Reset() error {
	_, err := e.DB.Exec(`TRUNCATE TABLE apps CASCADE`)
	return err
}

// Migrate runs the migrations.
func Migrate(db, path string) ([]error, bool) {
	return migrate.UpSync(db, path)
}
