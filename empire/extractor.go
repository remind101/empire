package empire

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/fsouza/go-dockerclient"
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

// fakeExtractor is a fake implementation of the Extractor interface.
type fakeExtractor struct{}

// Extract implements Extractor Extract.
func (e *fakeExtractor) Extract(image Image) (CommandMap, error) {
	pm := make(CommandMap)

	// Just return some fake processes.
	pm[ProcessType("web")] = Command("./bin/web")

	return pm, nil
}

type cmdExtractor struct {
	// Client is the docker client to use to pull the container image.
	client *docker.Client
}

func (e *cmdExtractor) Extract(image Image) (CommandMap, error) {
	pm := make(CommandMap)

	i, err := e.client.InspectImage(image.String())
	if err != nil {
		return pm, err
	}

	pm[ProcessType("web")] = Command(strings.Join(i.Config.Cmd, " "))

	return pm, nil
}

// procfileFallbackExtractor attempts to extract commands using the procfileExtractor.
// If that fails because Procfile does not exist, it uses the cmdExtractor instead.
type procfileFallbackExtractor struct {
	pe *procfileExtractor
	ce *cmdExtractor
}

func newProcfileFallbackExtractor(c *docker.Client) Extractor {
	return &procfileFallbackExtractor{
		pe: &procfileExtractor{
			client: c,
		},
		ce: &cmdExtractor{
			client: c,
		},
	}
}

func (e *procfileFallbackExtractor) Extract(image Image) (CommandMap, error) {
	cm, err := e.pe.Extract(image)
	// If err is a ProcfileError, Procfile doesn't exist.
	if _, ok := err.(*ProcfileError); ok {
		cm, err = e.ce.Extract(image)
	}

	return cm, err
}

// procfileExtractor is an implementation of the Extractor interface that can
// pull a docker image and extract its Procfile into a process.CommandMap.
type procfileExtractor struct {
	// Client is the docker client to use to pull the container image.
	client *docker.Client
}

// Extract implements Extractor Extract.
func (e *procfileExtractor) Extract(image Image) (CommandMap, error) {
	pm := make(CommandMap)

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
func (e *procfileExtractor) procfile(id string) (string, error) {
	p := ""

	c, err := e.client.InspectContainer(id)
	if err != nil {
		return "", err
	}

	if c.Config != nil {
		p = c.Config.WorkingDir
	}

	return path.Join(p, Procfile), nil
}

// createContainer creates a new docker container for the given docker image.
func (e *procfileExtractor) createContainer(i Image) (*docker.Container, error) {
	return e.client.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: i.String(),
		},
	})
}

// removeContainer removes a container by its ID.
func (e *procfileExtractor) removeContainer(containerID string) error {
	return e.client.RemoveContainer(docker.RemoveContainerOptions{
		ID: containerID,
	})
}

// copyFile copies a file from a container.
func (e *procfileExtractor) copyFile(containerID, path string) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.client.CopyFromContainer(docker.CopyFromContainerOptions{
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
