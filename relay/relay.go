package relay

import (
	"net"
	"sync"

	"github.com/fsouza/go-dockerclient"

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

type Options struct {
	Host             string
	SessionGenerator func() string
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
		Host:         options.Host,
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

func (r *Relay) CreateContainer(c *Container) error {
	if err := r.runner.Pull(c); err != nil {
		return err
	}
	return r.runner.Run(c)
}

func (r *Relay) AttachToContainer(name string, conn net.Conn) error {
	return r.runner.Attach(name, conn, conn)
}
