package empire // import "github.com/remind101/empire"

import (
	"github.com/mattes/migrate/migrate"
	"github.com/remind101/empire/scheduler"
)

// DefaultOptions is a default Options instance that can be passed when
// intializing a new Empire.
var DefaultOptions = Options{}

// DockerOptions is a set of options to configure a docker api client.
type DockerOptions struct {
	// The unix socket to connect to the docker api.
	Socket string

	// The docker registry to pull container images from.
	Registry string

	// Path to a certificate to use for TLS connections.
	CertPath string
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

	// Database connection string.
	DB string
}

// Empire is a context object that contains a collection of services.
type Empire struct {
	DB DB

	AppsService
	ConfigsService
	DeploysService
	Manager
	ReleasesService
	SlugsService
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

	slugsRepository, err := NewSlugsRepository(db)
	if err != nil {
		return nil, err
	}

	extractor, err := NewExtractor(
		options.Docker.Socket,
		options.Docker.Registry,
		options.Docker.CertPath,
	)
	if err != nil {
		return nil, err
	}

	appsRepository, err := NewAppsRepository(db)
	if err != nil {
		return nil, err
	}

	configsRepository, err := NewConfigsRepository(db)
	if err != nil {
		return nil, err
	}

	processesRepository, err := NewProcessesRepository(db)
	if err != nil {
		return nil, err
	}

	releasesRepository, err := NewReleasesRepository(db)
	if err != nil {
		return nil, err
	}

	jobsRepository, err := NewJobsRepository(db)
	if err != nil {
		return nil, err
	}

	apps, err := NewAppsService(appsRepository)
	if err != nil {
		return nil, err
	}

	configs, err := NewConfigsService(configsRepository)
	if err != nil {
		return nil, err
	}

	manager, err := NewManager(jobsRepository, scheduler)
	if err != nil {
		return nil, err
	}

	releases, err := NewReleasesService(
		releasesRepository,
		processesRepository,
		manager,
	)
	if err != nil {
		return nil, err
	}

	slugs, err := NewSlugsService(slugsRepository, extractor)
	if err != nil {
		return nil, err
	}

	deploys, err := NewDeploysService(
		options,
		apps,
		configs,
		slugs,
		releases,
	)
	if err != nil {
		return nil, err
	}

	return &Empire{
		DB:              db,
		AppsService:     apps,
		ConfigsService:  configs,
		DeploysService:  deploys,
		Manager:         manager,
		SlugsService:    slugs,
		ReleasesService: releases,
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
