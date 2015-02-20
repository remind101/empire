package empire // import "github.com/remind101/empire"

import (
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/formations"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/slugs"
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
}

// Empire is a context object that contains a collection of services.
type Empire struct {
	appsService       *apps.Service
	configsService    *configs.Service
	deploysService    DeploysService
	formationsService *formations.Service
	manager           Manager
	releasesService   ReleasesService
	slugsService      SlugsService
}

// New returns a new Empire instance.
func New(options Options) (*Empire, error) {
	manager, err := newManager(options)
	if err != nil {
		return nil, err
	}

	slugs, err := newSlugsService(options)
	if err != nil {
		return nil, err
	}

	return &Empire{
		manager:      manager,
		slugsService: slugs,
	}, nil
}

func (e *Empire) AppsService() *apps.Service {
	if e.appsService == nil {
		e.appsService = apps.NewService(nil)
	}

	return e.appsService
}

func (e *Empire) ConfigsService() *configs.Service {
	if e.configsService == nil {
		e.configsService = configs.NewService(nil)
	}

	return e.configsService
}

func (e *Empire) DeploysService() DeploysService {
	if e.deploysService == nil {
		e.deploysService = &deploysService{
			AppsService:     e.AppsService(),
			ConfigsService:  e.ConfigsService(),
			Manager:         e.Manager(),
			SlugsService:    e.SlugsService(),
			ReleasesService: e.ReleasesService(),
		}
	}

	return e.deploysService
}

func (e *Empire) FormationsService() *formations.Service {
	if e.formationsService == nil {
		e.formationsService = formations.NewService(nil)
	}

	return e.formationsService
}

func (e *Empire) Manager() Manager {
	if e.manager == nil {
		e.manager = NewManager(nil)
	}

	return e.manager
}

func (e *Empire) ReleasesService() ReleasesService {
	if e.releasesService == nil {
		e.releasesService = NewReleasesService(
			nil,
			e.FormationsService(),
		)
	}

	return e.releasesService
}

func (e *Empire) SlugsService() SlugsService {
	if e.slugsService == nil {
		e.slugsService = NewSlugsService(nil, nil)
	}

	return e.slugsService
}

func newSlugsService(options Options) (SlugsService, error) {
	r, err := slugs.NewRepository()
	if err != nil {
		return nil, err
	}

	e, err := slugs.NewExtractor(
		options.Docker.Socket,
		options.Docker.Registry,
		options.Docker.CertPath,
	)
	if err != nil {
		return nil, err
	}

	return NewSlugsService(r, e), nil
}

func newManager(options Options) (Manager, error) {
	s, err := scheduler.NewScheduler(options.Fleet.API)
	if err != nil {
		return nil, err
	}

	return NewManager(s), nil
}
