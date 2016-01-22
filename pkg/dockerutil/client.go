package dockerutil

import (
	"fmt"
	"os"

	"golang.org/x/net/context"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/dockerauth"
	"github.com/remind101/pkg/trace"
)

// NewDockerClient returns a new docker.Client using the given socket and certificate path.
func NewDockerClient(socket, certPath string) (*docker.Client, error) {
	if certPath != "" {
		cert := certPath + "/cert.pem"
		key := certPath + "/key.pem"
		ca := certPath + "/ca.pem"
		return docker.NewTLSClient(socket, cert, key, ca)
	}

	return docker.NewClient(socket)
}

// NewDockerClientFromEnv returns a new docker client configured by the DOCKER_*
// environment variables.
func NewDockerClientFromEnv() (*docker.Client, error) {
	return NewDockerClient(os.Getenv("DOCKER_HOST"), os.Getenv("DOCKER_CERT_PATH"))
}

// Client wraps a docker.Client to authenticate pulls.
type Client struct {
	*docker.Client

	// AuthProvider is the dockerauth.AuthProvider that will be used for pulling
	// images.
	AuthProvider dockerauth.AuthProvider
}

// NewClient returns a new Client instance.
func NewClient(authProvider dockerauth.AuthProvider, socket, certPath string) (*Client, error) {
	c, err := NewDockerClient(socket, certPath)
	if err != nil {
		return nil, err
	}
	return newClient(authProvider, c), nil
}

// NewClientFromEnv returns a new Client instance configured by the DOCKER_*
// environment variables.
func NewClientFromEnv(authProvider dockerauth.AuthProvider) (*Client, error) {
	c, err := NewDockerClientFromEnv()
	if err != nil {
		return nil, err
	}
	return newClient(authProvider, c), nil
}

func newClient(authProvider dockerauth.AuthProvider, c *docker.Client) *Client {
	if authProvider == nil {
		authProvider = dockerauth.NewMultiAuthProvider()
	}
	return &Client{AuthProvider: authProvider, Client: c}
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

	ctx, done := trace.Trace(ctx)
	err = c.Client.PullImage(opts, authConf)
	done(err, "PullImage", "registry", opts.Registry, "repository", opts.Repository, "tag", opts.Tag)
	return err
}

func (c *Client) CreateContainer(ctx context.Context, opts docker.CreateContainerOptions) (*docker.Container, error) {
	ctx, done := trace.Trace(ctx)
	container, err := c.Client.CreateContainer(opts)
	done(err, "CreateContainer", "image", opts.Config.Image)
	return container, err
}

func (c *Client) StartContainer(ctx context.Context, id string, config *docker.HostConfig) error {
	ctx, done := trace.Trace(ctx)
	err := c.Client.StartContainer(id, config)
	done(err, "StartContainer", "id", id)
	return err
}

func (c *Client) AttachToContainer(ctx context.Context, opts docker.AttachToContainerOptions) error {
	ctx, done := trace.Trace(ctx)
	err := c.Client.AttachToContainer(opts)
	done(err, "AttachToContainer", "container", opts.Container)
	return err
}

func (c *Client) StopContainer(ctx context.Context, id string, timeout uint) error {
	ctx, done := trace.Trace(ctx)
	err := c.Client.StopContainer(id, timeout)
	done(err, "StopContainer", "id", id, "timeout", timeout)
	return err
}

func (c *Client) RemoveContainer(ctx context.Context, opts docker.RemoveContainerOptions) error {
	ctx, done := trace.Trace(ctx)
	err := c.Client.RemoveContainer(opts)
	done(err, "RemoveContainer", "id", opts.ID)
	return err
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
