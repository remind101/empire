package relay // import "github.com/remind101/empire/relay"

import (
	"net"
	"strings"
	"sync"

	"github.com/fsouza/go-dockerclient"
	"golang.org/x/net/context"

	"code.google.com/p/go-uuid/uuid"
)

var (
	DefaultSessionGenerator = func() string { return uuid.New() }

	// DefaultOptions is a default Options instance that can be passed when
	// intializing a new Relay.
	DefaultOptions = Options{SessionGenerator: DefaultSessionGenerator}
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

type TcpOptions struct {
	Host string // Host that the tcp server is running on.
	Port string // Port that the tcp server is running on.
}

type Options struct {
	SessionGenerator func() string
	Tcp              TcpOptions
	Docker           DockerOptions
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

	// The rendezvous host
	Host string

	runner ContainerRunner

	genSessionId func() string
	sessions     map[string]bool
}

// New returns a new Relay instance.
func New(options Options) *Relay {
	sg := options.SessionGenerator
	if sg == nil {
		sg = DefaultSessionGenerator
	}

	var runner ContainerRunner
	var err error
	if options.Docker.Socket == "fake" {
		runner = &fakeRunner{}
	} else {
		runner, err = newDockerRunner(options.Docker.Socket, options.Docker.CertPath, options.Docker.Auth)
		if err != nil {
			panic(err)
		}
	}

	return &Relay{
		Host:         strings.Join([]string{options.Tcp.Host, options.Tcp.Port}, ":"),
		runner:       runner,
		genSessionId: sg,
		sessions:     map[string]bool{},
	}
}

func (r *Relay) NewSession() string {
	r.Lock()
	defer r.Unlock()
	id := r.genSessionId()
	r.sessions[id] = true
	return id
}

func (r *Relay) CreateContainer(ctx context.Context, c *Container) error {
	if err := r.runner.Pull(c); err != nil {
		return err
	}
	return r.runner.Run(c)
}

func (r *Relay) AttachToContainer(ctx context.Context, name string, conn net.Conn) error {
	return r.runner.Attach(name, conn, conn)
}
