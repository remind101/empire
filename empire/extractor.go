package empire

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/empire/pkg/registry"
	"gopkg.in/yaml.v2"
)

var (
	// Procfile is the name of the Procfile file.
	Procfile = "Procfile"
)

// Extractor represents an object that can extract the process types from an
// image.
type Extractor interface {
	// Extract takes a repo in the form `remind101/r101-api`, and an image
	// id, and extracts the process types from the image.
	Extract(Image) (CommandMap, error)
}

// NewExtractor returns a new Extractor instance.
func NewExtractor(socket, certPath string, auth *docker.AuthConfigurations) (Extractor, error) {
	if socket == "" {
		return newExtractor(), nil
	}

	c, err := newDockerClient(socket, certPath)
	if err != nil {
		return nil, err
	}

	return &ProcfileExtractor{
		Client:             c,
		AuthConfigurations: auth,
	}, nil
}

// extractor is a fake implementation of the Extractor interface.
type extractor struct{}

func newExtractor() *extractor {
	return &extractor{}
}

// Extract implements Extractor Extract.
func (e *extractor) Extract(image Image) (CommandMap, error) {
	pm := make(CommandMap)

	// Just return some fake processes.
	pm[ProcessType("web")] = Command("./bin/web")

	return pm, nil
}

// ProcfileExtractor is an implementation of the Extractor interface that can
// pull a docker image and extract it's Procfile into a process.CommandMap.
type ProcfileExtractor struct {
	// Client is the docker client to use to pull the container image.
	Client interface {
		PullImage(docker.PullImageOptions, docker.AuthConfiguration) error
		InspectContainer(string) (*docker.Container, error)
		CreateContainer(docker.CreateContainerOptions) (*docker.Container, error)
		RemoveContainer(docker.RemoveContainerOptions) error
		CopyFromContainer(docker.CopyFromContainerOptions) error
	}

	// AuthConfiguration contains the docker AuthConfiguration.
	*docker.AuthConfigurations
}

// Extract implements Extractor Extract.
func (e *ProcfileExtractor) Extract(image Image) (CommandMap, error) {
	pm := make(CommandMap)

	if err := e.pullImage(image); err != nil {
		return pm, err
	}

	c, err := e.createContainer(image)
	if err != nil {
		return pm, err
	}

	defer e.removeContainer(c.ID)

	procfile, err := e.procfile(c.ID)
	if err != nil {
		return pm, err
	}

	b, err := e.copyFile(c.ID, procfile)
	if err != nil {
		return pm, &ProcfileError{Err: err}
	}

	return ParseProcfile(b)
}

// procfile returns the path to the Procfile. If the container has a WORKDIR
// set, then this will return a path to the Procfile within that directory.
func (e *ProcfileExtractor) procfile(id string) (string, error) {
	p := ""

	c, err := e.Client.InspectContainer(id)
	if err != nil {
		return "", err
	}

	if c.Config != nil {
		p = c.Config.WorkingDir
	}

	return path.Join(p, Procfile), nil
}

// pullImage can pull a docker image from a repo, by it's imageID.
//
// Because docker does not support pulling an image by ID, we're assuming that
// the docker image has been tagged with it's own ID beforehand.
func (e *ProcfileExtractor) pullImage(i Image) error {
	var a docker.AuthConfiguration

	reg, _, err := registry.Split(string(i.Repo))
	if err != nil {
		return err
	}

	if reg == "" {
		reg = "https://index.docker.io/v1/"
	}

	if c, ok := e.Configs[reg]; ok {
		a = c
	}

	return e.Client.PullImage(docker.PullImageOptions{
		Repository:   string(i.Repo),
		Tag:          i.Tag,
		OutputStream: os.Stdout,
	}, a)
}

// createContainer creates a new docker container for the given docker image.
func (e *ProcfileExtractor) createContainer(i Image) (*docker.Container, error) {
	return e.Client.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: i.String(),
		},
	})
}

// removeContainer removes a container by it's ID.
func (e *ProcfileExtractor) removeContainer(containerID string) error {
	return e.Client.RemoveContainer(docker.RemoveContainerOptions{
		ID: containerID,
	})
}

// copyFile copies a file from a container.
func (e *ProcfileExtractor) copyFile(containerID, path string) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.Client.CopyFromContainer(docker.CopyFromContainerOptions{
		Container:    containerID,
		Resource:     path,
		OutputStream: &buf,
	}); err != nil {
		return nil, err
	}

	// Open the tar archive for reading.
	r := bytes.NewReader(buf.Bytes())

	return firstFile(tar.NewReader(r))
}

// dockerClient is a fake docker client that can be used with the
// ProcfileExtractor.
type dockerClient struct {
	procfile string
}

func (c *dockerClient) PullImage(options docker.PullImageOptions, auth docker.AuthConfiguration) error {
	return nil
}

func (c *dockerClient) CreateContainer(options docker.CreateContainerOptions) (*docker.Container, error) {
	return &docker.Container{}, nil
}

func (c *dockerClient) RemoveContainer(options docker.RemoveContainerOptions) error {
	return nil
}

func (c *dockerClient) CopyFromContainer(options docker.CopyFromContainerOptions) error {
	pf := c.procfile
	tw := tar.NewWriter(options.OutputStream)
	if err := tw.WriteHeader(&tar.Header{
		Name: "/Procfile",
		Size: int64(len(pf)),
	}); err != nil {
		return err
	}

	if _, err := tw.Write([]byte(pf)); err != nil {
		return err
	}

	return nil
}

// Example instance: Procfile doesn't exist
type ProcfileError struct {
	Err error
}

func (e *ProcfileError) Error() string {
	return fmt.Sprintf("Procfile not found: %s", e.Err)
}

// ParseProcfile takes a byte slice representing a YAML Procfile and parses it
// into a processes.CommandMap.
func ParseProcfile(b []byte) (CommandMap, error) {
	pm := make(CommandMap)

	if err := yaml.Unmarshal(b, &pm); err != nil {
		return pm, err
	}

	return pm, nil
}

// firstFile extracts the first file from a tar archive.
func firstFile(tr *tar.Reader) ([]byte, error) {
	if _, err := tr.Next(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, tr); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func newDockerClient(socket, certPath string) (*docker.Client, error) {
	if certPath != "" {
		cert := certPath + "/cert.pem"
		key := certPath + "/key.pem"
		ca := ""
		return docker.NewTLSClient(socket, cert, key, ca)
	}

	return docker.NewClient(socket)
}
