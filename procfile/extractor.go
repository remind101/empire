package procfile

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/remind101/empire/pkg/image"

	"github.com/fsouza/go-dockerclient"
)

var (
	// ProcfileName is the name of the Procfile file.
	ProcfileName = "Procfile"
)

// Extract represents something that can extract a Procfile from an image.
type Extractor interface {
	Extract(image.Image) (Procfile, error)
}

type ExtractorFunc func(image.Image) (Procfile, error)

func (fn ExtractorFunc) Extract(image image.Image) (Procfile, error) {
	return fn(image)
}

// CommandExtractor is an Extractor implementation that returns a Procfile based
// on the CMD directive in the Dockerfile. It makes the assumption that the cmd
// is a "web" process.
type CMDExtractor struct {
	// Client is the docker client to use to pull the container image.
	client *docker.Client
}

func NewCMDExtractor(c *docker.Client) *CMDExtractor {
	return &CMDExtractor{client: c}
}

func (e *CMDExtractor) Extract(img image.Image) (Procfile, error) {
	pm := make(Procfile)

	i, err := e.client.InspectImage(img.String())
	if err != nil {
		return pm, err
	}

	pm["web"] = strings.Join(i.Config.Cmd, " ")

	return pm, nil
}

// MultiExtractor is an Extractor implementation that tries multiple Extractors
// in succession until one succeeds.
func MultiExtractor(extractors ...Extractor) Extractor {
	return ExtractorFunc(func(image image.Image) (Procfile, error) {
		for _, extractor := range extractors {
			p, err := extractor.Extract(image)

			// Yay!
			if err == nil {
				return p, nil
			}

			// Try the next one
			if _, ok := err.(*ProcfileError); ok {
				continue
			}

			// Bubble up the error
			return p, err
		}

		return nil, &ProcfileError{
			Err: errors.New("no suitable Procfile extractor found"),
		}
	})
}

// FileExtractor is an implementation of the Extractor interface that extracts
// the Procfile from the images WORKDIR.
type FileExtractor struct {
	// Client is the docker client to use to pull the container image.
	client *docker.Client
}

func NewFileExtractor(c *docker.Client) *FileExtractor {
	return &FileExtractor{client: c}
}

// Extract implements Extractor Extract.
func (e *FileExtractor) Extract(img image.Image) (Procfile, error) {
	pm := make(Procfile)

	c, err := e.createContainer(img)
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
func (e *FileExtractor) procfile(id string) (string, error) {
	p := ""

	c, err := e.client.InspectContainer(id)
	if err != nil {
		return "", err
	}

	if c.Config != nil {
		p = c.Config.WorkingDir
	}

	return path.Join(p, ProcfileName), nil
}

// createContainer creates a new docker container for the given docker image.
func (e *FileExtractor) createContainer(img image.Image) (*docker.Container, error) {
	return e.client.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: img.String(),
		},
	})
}

// removeContainer removes a container by its ID.
func (e *FileExtractor) removeContainer(containerID string) error {
	return e.client.RemoveContainer(docker.RemoveContainerOptions{
		ID: containerID,
	})
}

// copyFile copies a file from a container.
func (e *FileExtractor) copyFile(containerID, path string) ([]byte, error) {
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
