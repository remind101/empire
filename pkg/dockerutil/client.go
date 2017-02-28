package dockerutil

import (
	"fmt"

	"golang.org/x/net/context"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/dockerauth"
)

// The /containers/{name:.*}/copy endpoint was removed in this version of the
// Docker API.
var dockerAPI124, _ = docker.NewAPIVersion("1.24")

// NewDockerClient returns a new docker.Client using the given host and certificate path.
func NewDockerClient(host, certPath string) (*docker.Client, error) {
	if certPath != "" {
		cert := certPath + "/cert.pem"
		key := certPath + "/key.pem"
		ca := certPath + "/ca.pem"
		return docker.NewTLSClient(host, cert, key, ca)
	}

	return docker.NewClient(host)
}

// Client wraps a docker.Client to authenticate pulls.
type Client struct {
	*docker.Client

	// AuthProvider is the dockerauth.AuthProvider that will be used for pulling
	// images.
	AuthProvider dockerauth.AuthProvider

	apiVersion docker.APIVersion
}

// NewClient returns a new Client instance.
func NewClient(authProvider dockerauth.AuthProvider, host, certPath string) (*Client, error) {
	c, err := NewDockerClient(host, certPath)
	if err != nil {
		return nil, err
	}
	return newClient(authProvider, c)
}

func newClient(authProvider dockerauth.AuthProvider, c *docker.Client) (*Client, error) {
	if authProvider == nil {
		authProvider = dockerauth.NewMultiAuthProvider()
	}

	env, err := c.Version()
	if err != nil {
		return nil, fmt.Errorf("error getting Docker version: %v", err)
	}

	apiVersion, err := docker.NewAPIVersion(env.Get("ApiVersion"))
	if err != nil {
		return nil, err
	}

	return &Client{
		AuthProvider: authProvider,
		Client:       c,
		apiVersion:   apiVersion,
	}, nil
}

// PullImage wraps the docker clients PullImage to handle authentication.
func (c *Client) PullImage(ctx context.Context, opts docker.PullImageOptions) error {
	// This is to workaround an issue in the Docker API, where it doesn't
	// respect the registry param. We have to put the registry in the
	// repository field.
	if opts.Registry != "" {
		opts.Repository = fmt.Sprintf("%s/%s", opts.Registry, opts.Repository)
	}

	authConf, err := authConfiguration(c.AuthProvider, opts.Registry)
	if err != nil {
		return err
	}

	return c.Client.PullImage(opts, authConf)
}

func (c *Client) CreateContainer(ctx context.Context, opts docker.CreateContainerOptions) (*docker.Container, error) {
	return c.Client.CreateContainer(opts)
}

func (c *Client) StartContainer(ctx context.Context, id string, config *docker.HostConfig) error {
	return c.Client.StartContainer(id, config)
}

func (c *Client) AttachToContainer(ctx context.Context, opts docker.AttachToContainerOptions) error {
	return c.Client.AttachToContainer(opts)
}

func (c *Client) StopContainer(ctx context.Context, id string, timeout uint) error {
	return c.Client.StopContainer(id, timeout)
}

func (c *Client) RemoveContainer(ctx context.Context, opts docker.RemoveContainerOptions) error {
	return c.Client.RemoveContainer(opts)
}

func (c *Client) CopyFromContainer(ctx context.Context, options docker.CopyFromContainerOptions) error {
	if c.apiVersion.GreaterThanOrEqualTo(dockerAPI124) {
		return c.Client.DownloadFromContainer(options.Container, docker.DownloadFromContainerOptions{
			OutputStream: options.OutputStream,
			Path:         options.Resource,
		})
	}

	return c.Client.CopyFromContainer(options)
}

func authConfiguration(provider dockerauth.AuthProvider, registry string) (docker.AuthConfiguration, error) {
	authConf, err := provider.AuthConfiguration(registry)
	if err != nil {
		return docker.AuthConfiguration{}, err
	}

	if authConf != nil {
		return *authConf, nil
	}

	return docker.AuthConfiguration{}, nil
}
