package dockerutil

import (
	"os"

	"github.com/fsouza/go-dockerclient"
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

	// Auth is the docker.AuthConfiguration that will be used for pulling
	// images.
	Auth *docker.AuthConfigurations
}

// NewClient returns a new Client instance.
func NewClient(auth *docker.AuthConfigurations, socket, certPath string) (*Client, error) {
	c, err := NewDockerClient(socket, certPath)
	if err != nil {
		return nil, err
	}
	return newClient(auth, c), nil
}

// NewClientFromEnv returns a new Client instance configured by the DOCKER_*
// environment variables.
func NewClientFromEnv(auth *docker.AuthConfigurations) (*Client, error) {
	c, err := NewDockerClientFromEnv()
	if err != nil {
		return nil, err
	}
	return newClient(auth, c), nil
}

func newClient(auth *docker.AuthConfigurations, c *docker.Client) *Client {
	if auth == nil {
		auth = &docker.AuthConfigurations{}
	}
	return &Client{Auth: auth, Client: c}
}

// PullImage wraps the docker clients PullImage to handle authentication.
func (c *Client) PullImage(opts docker.PullImageOptions) error {
	var a docker.AuthConfiguration

	reg := opts.Registry

	if reg == "" {
		reg = "https://index.docker.io/v1/"
	}

	if c, ok := c.Auth.Configs[reg]; ok {
		a = c
	}

	return c.Client.PullImage(opts, a)
}
