package relay

import (
	"errors"
	"os"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

// ErrInvalidRepo is returned by Split when the repo is not a valid repo.
var ErrInvalidRepo = errors.New("registry: not a valid docker repo")

type ContainerRunner interface {
	Pull(*Container) error
	Run(*Container) error
}

type fakeRunner struct {
}

func (f *fakeRunner) Pull(c *Container) error {
	return nil
}

func (f *fakeRunner) Run(c *Container) error {
	return nil
}

type dockerRunner struct {
	client *docker.Client
	auth   *docker.AuthConfigurations
}

func newDockerRunner(socket, certPath string, auth *docker.AuthConfigurations) (*dockerRunner, error) {
	var err error
	var dc *docker.Client

	switch {
	case certPath != "":
		cert := certPath + "/cert.pem"
		key := certPath + "/key.pem"
		ca := ""
		dc, err = docker.NewTLSClient(socket, cert, key, ca)
	case socket != "":
		dc, err = docker.NewClient(socket)
	}

	if err != nil {
		return nil, err
	}

	return &dockerRunner{
		client: dc,
		auth:   auth,
	}, nil
}

func (d *dockerRunner) Pull(c *Container) error {
	return nil
}

func (d *dockerRunner) Run(c *Container) error {
	return nil
}

func (d *dockerRunner) pullImage(image string, tag string) error {
	var a docker.AuthConfiguration

	reg, _, err := d.splitRepo(image)
	if err != nil {
		return err
	}

	if reg == "" {
		reg = "https://index.docker.io/v1/"
	}

	if c, ok := d.auth.Configs[reg]; ok {
		a = c
	}

	return d.client.PullImage(docker.PullImageOptions{
		Repository:   image,
		Tag:          tag,
		OutputStream: os.Stdout,
	}, a)
}

// Split splits a full docker repo into registry and path segments.
func (d *dockerRunner) splitRepo(fullRepo string) (registry string, path string, err error) {
	parts := strings.Split(fullRepo, "/")

	if len(parts) < 2 {
		return "", "", ErrInvalidRepo
	}

	if len(parts) == 2 {
		return "", strings.Join(parts, "/"), nil
	}

	return parts[0], strings.Join(parts[1:], "/"), nil
}
