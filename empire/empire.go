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
	DB DB

	AccessTokensService
	AppsService
	ConfigsService
	DeploysService
	JobsService
	JobStatesService
	Manager
	ReleasesService
	SlugsService
	ProcessesService
}

// New returns a new Empire instance.
func New(options Options) (*Empire, error) {
	db, err := NewDB(options.DB)
	if err != nil {
		return nil, err
	}

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

	apps := &appsService{
		DB:          db,
		JobsService: jobs,
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

	deploys := &deploysService{
		AppsService:     apps,
		ConfigsService:  configs,
		SlugsService:    slugs,
		ReleasesService: releases,
	}

	return &Empire{
		DB:                  db,
		AccessTokensService: accessTokens,
		AppsService:         apps,
		ConfigsService:      configs,
		DeploysService:      deploys,
		JobsService:         jobs,
		JobStatesService:    jobStates,
		Manager:             manager,
		SlugsService:        slugs,
		ReleasesService:     releases,
		ProcessesService:    processes,
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
	if fleetURL == "" {
		return container.NewFakeScheduler(), nil
	}

	u, err := url.Parse(fleetURL)
	if err != nil {
		return nil, err
	}

	return container.NewFleetScheduler(u)
}
