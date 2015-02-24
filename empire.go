package empire // import "github.com/remind101/empire"

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
	AppsService
	ConfigsService
	DeploysService
	Manager
	ReleasesService
	SlugsService
}

// New returns a new Empire instance.
func New(options Options) (*Empire, error) {
	apps, err := NewAppsService(options)
	if err != nil {
		return nil, err
	}

	configs, err := NewConfigsService(options)
	if err != nil {
		return nil, err
	}

	manager, err := NewManager(options)
	if err != nil {
		return nil, err
	}

	releases, err := NewReleasesService(options, manager)
	if err != nil {
		return nil, err
	}

	slugs, err := NewSlugsService(options)
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
		AppsService:     apps,
		ConfigsService:  configs,
		DeploysService:  deploys,
		Manager:         manager,
		SlugsService:    slugs,
		ReleasesService: releases,
	}, nil
}
