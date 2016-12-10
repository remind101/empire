package dockerutil

import (
	"fmt"
	"os"

	"golang.org/x/net/context"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/dockerauth"
	"github.com/remind101/empire/tracer"
)

// The /containers/{name:.*}/copy endpoint was removed in this version of the
// Docker API.
var dockerAPI124, _ = docker.NewAPIVersion("1.24")

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

	apiVersion docker.APIVersion
}

// NewClient returns a new Client instance.
func NewClient(authProvider dockerauth.AuthProvider, socket, certPath string) (*Client, error) {
	c, err := NewDockerClient(socket, certPath)
	if err != nil {
		return nil, err
	}
	return newClient(authProvider, c)
}

// NewClientFromEnv returns a new Client instance configured by the DOCKER_*
// environment variables.
func NewClientFromEnv(authProvider dockerauth.AuthProvider) (*Client, error) {
	c, err := NewDockerClientFromEnv()
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

func (c *Client) newSpan(ctx context.Context, method string) *tracer.Span {
	span := tracer.NewChildSpanFromContext(method, ctx)
	span.Service = "docker"
	span.Resource = method
	return span
}

// PullImage wraps the docker clients PullImage to handle authentication.
func (c *Client) PullImage(ctx context.Context, opts docker.PullImageOptions) error {
	span := c.newSpan(ctx, "PullImage")
	err := c.pullImage(opts)
	span.FinishWithErr(err)
	return err
}

func (c *Client) pullImage(opts docker.PullImageOptions) error {
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
	span := c.newSpan(ctx, "CreateContainer")
	container, err := c.Client.CreateContainer(opts)
	span.FinishWithErr(err)
	return container, err
}

func (c *Client) StartContainer(ctx context.Context, id string, config *docker.HostConfig) error {
	span := c.newSpan(ctx, "StartContainer")
	err := c.Client.StartContainer(id, config)
	span.FinishWithErr(err)
	return err
}

func (c *Client) AttachToContainer(ctx context.Context, opts docker.AttachToContainerOptions) error {
	span := c.newSpan(ctx, "AttachToContainer")
	err := c.Client.AttachToContainer(opts)
	span.FinishWithErr(err)
	return err
}

func (c *Client) StopContainer(ctx context.Context, id string, timeout uint) error {
	span := c.newSpan(ctx, "StopContainer")
	err := c.Client.StopContainer(id, timeout)
	span.FinishWithErr(err)
	return err
}

func (c *Client) RemoveContainer(ctx context.Context, opts docker.RemoveContainerOptions) error {
	span := c.newSpan(ctx, "RemoveContainer")
	err := c.Client.RemoveContainer(opts)
	span.FinishWithErr(err)
	return err
}

func (c *Client) CopyFromContainer(ctx context.Context, options docker.CopyFromContainerOptions) error {
	span := c.newSpan(ctx, "CopyFromContainer")
	err := c.copyFromContainer(options)
	span.FinishWithErr(err)
	return err
}

func (c *Client) copyFromContainer(options docker.CopyFromContainerOptions) error {
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
