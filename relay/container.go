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

// RepoPattern matches strings like `quay.io/remind101/acme-inc:latest` or `remind101/acme-inc:latest`
var RepoPattern = regexp.MustCompile(`(\S+\/)?(\S+\/\S+):(\S+)`)

// ContainerManager defines an interface for managing containers.
type ContainerManager interface {
	Pull(*Container) error
	Create(*Container) error
	Attach(string, io.Reader, io.Writer) error
	Start(string) error
	Wait(string) (int, error)
	Stop(string) error
	Remove(string) error
}

// newContainerManager returns a ContainerManager based on the given options.
func newContainerManager(options DockerOptions) (manager ContainerManager) {
	var err error

	if options.Socket == "fake" {
		manager = &fakeManager{}
	} else {
		manager, err = NewDockerManager(options.Socket, options.CertPath, options.Auth)
		if err != nil {
			panic(err)
		}
	}
	return manager
}

type fakeManager struct {
}

func (f *fakeManager) Pull(c *Container) error {
	return nil
}

func (f *fakeManager) Create(c *Container) error {
	return nil
}

func (f *fakeManager) Attach(name string, input io.Reader, output io.Writer) error {
	return nil
}

func (f *fakeManager) Start(name string) error {
	return nil
}

func (f *fakeManager) Wait(name string) (int, error) {
	return 0, nil
}

func (f *fakeManager) Stop(name string) error {
	return nil
}

func (f *fakeManager) Remove(name string) error {
	return nil
}

type dockerManager struct {
	client *docker.Client
	auth   *docker.AuthConfigurations
}

func NewDockerManager(socket, certPath string, auth *docker.AuthConfigurations) (*dockerManager, error) {
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
		return nil, errors.New("NewDockerManager needs a socket or a certPath")
	}

	if err != nil {
		return nil, err
	}

	return &dockerManager{
		client: dc,
		auth:   auth,
	}, nil
}

func (d *dockerManager) Pull(c *Container) error {
	return d.pullImage(c.Image)
}

func (d *dockerManager) Create(c *Container) error {
	env := []string{}
	for k, v := range c.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	opts := docker.CreateContainerOptions{
		Name: c.Name,
		Config: &docker.Config{
			Tty:          c.Attach,
			AttachStdin:  c.Attach,
			AttachStdout: c.Attach,
			AttachStderr: c.Attach,
			OpenStdin:    c.Attach,

			Image: c.Image,
			Cmd:   []string{c.Command},
			Env:   env,
		},
		HostConfig: &docker.HostConfig{},
	}
	_, err := d.client.CreateContainer(opts)
	return err
}

func (d *dockerManager) Attach(name string, input io.Reader, output io.Writer) error {
	opts := docker.AttachToContainerOptions{
		Container:    name,
		InputStream:  input,
		OutputStream: output,
		ErrorStream:  output,
		Logs:         true,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  true,
	}

	return d.client.AttachToContainer(opts)
}

func (d *dockerManager) Start(name string) error {
	return d.client.StartContainer(name, nil)
}

func (d *dockerManager) Wait(name string) (int, error) {
	return d.client.WaitContainer(name)
}

func (d *dockerManager) Stop(name string) error {
	return d.client.StopContainer(name, 10)
}

func (d *dockerManager) Remove(name string) error {
	return d.client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    name,
		Force: true,
	})
}

func (d *dockerManager) pullImage(image string) error {
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

// parseRepo splits a full docker repo into registry, repo and tag segments.
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
