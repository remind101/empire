package slugs

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/images"
	"github.com/remind101/empire/processes"
	"github.com/remind101/empire/repos"
	"gopkg.in/yaml.v2"
)

// Extractor represents an object that can extract the process types from an
// image.
type Extractor interface {
	// Extract takes a repo in the form `remind101/r101-api`, and an image
	// id, and extracts the process types from the image.
	Extract(*images.Image) (ProcessMap, error)
}

// NewExtractor returns a new Extractor instance.
func NewExtractor(socket, registry, certPath string) (Extractor, error) {
	if socket == "" {
		return newExtractor(), nil
	}

	c, err := newDockerClient(socket, certPath)
	if err != nil {
		return nil, err
	}

	return &ProcfileExtractor{
		Registry: registry,
		Client:   c,
	}, nil
}

// extractor is a fake implementation of the Extractor interface.
type extractor struct{}

func newExtractor() *extractor {
	return &extractor{}
}

// Extract implements Extractor Extract.
func (e *extractor) Extract(image *images.Image) (ProcessMap, error) {
	pm := make(ProcessMap)

	// Just return some fake processes.
	pm[processes.Type("web")] = processes.Command("./bin/web")

	return pm, nil
}

// ProcfileExtractor is an implementation of the Extractor interface that can
// pull a docker image and extract it's Procfile into a ProcessMap.
type ProcfileExtractor struct {
	// Registry is the registry to use to pull the image from. The zero
	// value is the default docker registry.
	Registry string

	// Path is the path to the Procfile to extract. The zero value is
	// /Procfile.
	Path string

	// Client is the docker client to use to pull the container image.
	Client interface {
		PullImage(docker.PullImageOptions, docker.AuthConfiguration) error
		CreateContainer(docker.CreateContainerOptions) (*docker.Container, error)
		RemoveContainer(docker.RemoveContainerOptions) error
		CopyFromContainer(docker.CopyFromContainerOptions) error
	}

	// AuthConfiguration contains the docker AuthConfiguration.
	docker.AuthConfiguration
}

// Extract implements Extractor Extract.
func (e *ProcfileExtractor) Extract(image *images.Image) (ProcessMap, error) {
	pm := make(ProcessMap)

	repo := e.fullRepo(image.Repo)
	if err := e.pullImage(repo, image.ID); err != nil {
		return pm, err
	}

	c, err := e.createContainer(repo, image.ID)
	if err != nil {
		return pm, err
	}

	defer e.removeContainer(c.ID)

	b, err := e.copyFile(c.ID, e.path())
	if err != nil {
		return pm, &ProcfileError{Err: err}
	}

	return ParseProcfile(b)
}

// fullRepo returns the fully qualified docker repo. For example, the fully
// qualified path for `ejholmes/docker-statsd` on quay.io would be:
//
//	quay.io/ejholmes/docker-statsd
//
// But the fully qualified repo for the official docker registry is:
//
//	ejholmes/docker-statsd
func (e *ProcfileExtractor) fullRepo(repo repos.Repo) string {
	if e.Registry != "" {
		return e.Registry + "/" + string(repo)
	}

	return string(repo)
}

// path returns the path to the Procfile.
func (e *ProcfileExtractor) path() string {
	if e.Path != "" {
		return e.Path
	}

	return DefaultProcfilePath
}

// pullImage can pull a docker image from a repo, by it's imageID.
//
// Because docker does not support pulling an image by ID, we're assuming that
// the docker image has been tagged with it's own ID beforehand.
func (e *ProcfileExtractor) pullImage(repo, imageID string) error {
	return e.Client.PullImage(docker.PullImageOptions{
		Repository:   repo,
		Tag:          imageID,
		OutputStream: os.Stdout,
	}, e.AuthConfiguration)
}

// createContainer creates a new docker container for the given docker image.
func (e *ProcfileExtractor) createContainer(repo, imageID string) (*docker.Container, error) {
	return e.Client.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: repo + ":" + imageID,
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
// into a ProcessMap.
func ParseProcfile(b []byte) (ProcessMap, error) {
	pm := make(ProcessMap)

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
