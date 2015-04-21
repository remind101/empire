package relay // import "github.com/remind101/empire/relay"

import (
	"fmt"
	"io"
	"sync"

	"github.com/fsouza/go-dockerclient"
	"golang.org/x/net/context"

	"code.google.com/p/go-uuid/uuid"
)

var (
	// DefaultContainerNameFunc is the default implementation for generating container names.
	DefaultContainerNameFunc = func(s string) string { return fmt.Sprintf("%s.%s", s, uuid.New()) }
)

// DockerOptions is a set of options to configure a docker api client.
type DockerOptions struct {
	// The default docker organization to use.
	Organization string

	// The unix socket to connect to the docker api.
	Socket string

	// Path to a certificate to use for TLS connections.
	CertPath string

	// A set of docker registry credentials.
	Auth *docker.AuthConfigurations
}

// TcpOptions is a set of options to configure the tcp server.
type TcpOptions struct {
	// Host that the tcp server is running on.
	Host string

	// Port that the tcp server is running on.
	Port string
}

// Options is the main set of options to configure relay.
type Options struct {
	ContainerNameFunc func(string) string
	Tcp               TcpOptions
	Docker            DockerOptions
}

// Container represents a docker container to run.
type Container struct {
	Image     string            `json:"image"`
	Name      string            `json:"name"`
	Command   string            `json:"command"`
	State     string            `json:"state"`
	Env       map[string]string `json:"env"`
	Attach    bool              `json:"attach"`
	AttachURL string            `json:"attach_url"`
}

type Relay struct {
	sync.Mutex

	// The rendezvous host.
	Host string

	// The container manager.
	manager ContainerManager

	// The func to use to generate container names.
	containerNameFunc func(string) string

	// The map of container names to container structs.
	sessions map[string]*Container
}

// New returns a new Relay instance with sensible defaults.
func New(options Options) *Relay {
	return &Relay{
		Host:              fmt.Sprintf("%s:%s", options.Tcp.Host, options.Tcp.Port),
		manager:           newContainerManager(options.Docker),
		containerNameFunc: options.ContainerNameFunc,
		sessions:          map[string]*Container{},
	}
}

// GenContainerName generates a new container name.
func (r *Relay) GenContainerName(s string) string {
	if r.containerNameFunc != nil {
		return r.containerNameFunc(s)
	}
	return DefaultContainerNameFunc(s)
}

// RegisterContainer registers a new container, ready to be started over a TCP session.
func (r *Relay) RegisterContainer(name string, c *Container) {
	r.Lock()
	defer r.Unlock()
	r.sessions[name] = c
}

// UnregisterContainer unregisters a container.
func (r *Relay) UnregisterContainer(name string, c *Container) {
	r.Lock()
	defer r.Unlock()
	delete(r.sessions, name)
}

// CreateContainer creates a new container instance, but doesn't start it.
func (r *Relay) CreateContainer(ctx context.Context, c *Container) error {
	if err := r.manager.Pull(c); err != nil {
		return err
	}
	return r.manager.Create(c)
}

// AttachToContainer attaches IO to an existing container.
func (r *Relay) AttachToContainer(ctx context.Context, name string, in io.Reader, out io.Writer) error {
	return r.manager.Attach(name, in, out)
}

// StartContainer starts a container. This should be called after creating and attaching to a container.
func (r *Relay) StartContainer(ctx context.Context, name string) error {
	return r.manager.Start(name)
}

// WaitContainer blocks until a container has finished runnning.
func (r *Relay) WaitContainer(ctx context.Context, name string) (int, error) {
	return r.manager.Wait(name)
}
