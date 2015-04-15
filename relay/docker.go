package relay

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

// ErrInvalidRepo is returned by Split when the repo is not a valid repo.
var ErrInvalidRepo = errors.New("registry: not a valid docker repo")

type ContainerRunner interface {
	Pull(*Container) error
	Run(*Container) error
	Attach(string, io.Reader, io.Writer) error
}

type fakeRunner struct {
}

func (f *fakeRunner) Pull(c *Container) error {
	return nil
}

func (f *fakeRunner) Run(c *Container) error {
	return nil
}
func (f *fakeRunner) Attach(name string, input io.Reader, output io.Writer) error {
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
		cert := path.Join(certPath, "cert.pem")
		key := path.Join(certPath, "key.pem")
		ca := ""
		dc, err = docker.NewTLSClient(socket, cert, key, ca)
	case socket != "":
		dc, err = docker.NewClient(socket)
	default:
		return nil, errors.New("newDockerRunner needs a socket or a certPath")
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
	return d.pullImage(c.Image, "latest")
}

func (d *dockerRunner) Run(c *Container) error {
	env := []string{}
	for k, v := range c.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	opts := docker.CreateContainerOptions{
		Name: c.Name,
		Config: &docker.Config{
			Tty:   c.Attach,
			Image: c.Image,
			Cmd:   []string{c.Command},
			Env:   env,
		},
		HostConfig: &docker.HostConfig{},
	}
	d.client.CreateContainer(opts)
	return nil
}

func (d *dockerRunner) Attach(name string, input io.Reader, output io.Writer) error {
	opts := docker.AttachToContainerOptions{
		Container:    name,
		InputStream:  input,
		OutputStream: output,
		ErrorStream:  output,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  true,
	}

	return d.client.AttachToContainer(opts)
}

func (d *dockerRunner) pullImage(image string, tag string) error {
	var a docker.AuthConfiguration

	reg, _, err := splitRepo(image)
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
func splitRepo(fullRepo string) (registry string, path string, err error) {
	parts := strings.Split(fullRepo, "/")

	if len(parts) < 2 {
		return "", "", ErrInvalidRepo
	}

	if len(parts) == 2 {
		return "", strings.Join(parts, "/"), nil
	}

	return parts[0], strings.Join(parts[1:], "/"), nil
}
