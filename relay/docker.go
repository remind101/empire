package relay

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

// ErrInvalidRepo is returned by parseRepo when the repo is not a valid repo.
var ErrInvalidRepo = errors.New("registry: not a valid docker repo")
var RepoPattern = regexp.MustCompile(`(\S+\/)?(\S+\/\S+):(\S+)`)

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
	return d.pullImage(c.Image)
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

func (d *dockerRunner) pullImage(image string) error {
	var a docker.AuthConfiguration

	reg, repo, tag, err := parseRepo(image)
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
		Repository:   repo,
		Tag:          tag,
		OutputStream: os.Stdout,
	}, a)
}

// Split splits a full docker repo into registry, repo and tag segments.
//
// Examples:
//
//     quay.io/remind101/acme-inc:latest # => registry: "quay.io", repo: "remind101/acme-inc", tag: "latest"
//     remind101/acme-inc:latest         # => registry: "", repo: "remind101/acme-inc", tag: "latest"
func parseRepo(fullRepo string) (registry string, repo string, tag string, err error) {
	m := RepoPattern.FindStringSubmatch(fullRepo)
	if len(m) == 0 {
		return "", "", "", ErrInvalidRepo
	}

	// Registy subpattern was matched.
	if len(m) == 4 {
		return strings.TrimRight(m[1], "/"), m[2], m[3], nil
	}

	return "", m[1], m[2], nil
}
