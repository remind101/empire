package empire

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"

	"golang.org/x/net/context"

	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/procfile"

	"github.com/fsouza/go-dockerclient"
)

// Procfile is the name of the Procfile file that Empire will use to
// determine the process formation.
const Procfile = "Procfile"

// ProcfileExtractor represents something that can extract a Procfile from an image.
type ProcfileExtractor interface {
	Extract(context.Context, image.Image, io.Writer) ([]byte, error)
}

type ProcfileExtractorFunc func(context.Context, image.Image, io.Writer) ([]byte, error)

func (fn ProcfileExtractorFunc) Extract(ctx context.Context, image image.Image, w io.Writer) ([]byte, error) {
	return fn(ctx, image, w)
}

// cmdExtractor is an Extractor implementation that returns a Procfile based
// on the CMD directive in the Dockerfile. It makes the assumption that the cmd
// is a "web" process.
type cmdExtractor struct {
	// Client is the docker client to use to pull the container image.
	client *docker.Client
}

func newCMDExtractor(c *docker.Client) *cmdExtractor {
	return &cmdExtractor{client: c}
}

func (e *cmdExtractor) Extract(_ context.Context, img image.Image, _ io.Writer) ([]byte, error) {
	i, err := e.client.InspectImage(img.String())
	if err != nil {
		return nil, err
	}

	return procfile.Marshal(procfile.ExtendedProcfile{
		"web": procfile.Process{
			Command: i.Config.Cmd,
		},
	})
}

// multiExtractor is an Extractor implementation that tries multiple Extractors
// in succession until one succeeds.
func multiExtractor(extractors ...ProcfileExtractor) ProcfileExtractor {
	return ProcfileExtractorFunc(func(ctx context.Context, image image.Image, w io.Writer) ([]byte, error) {
		for _, extractor := range extractors {
			p, err := extractor.Extract(ctx, image, w)

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

// fileExtractor is an implementation of the Extractor interface that extracts
// the Procfile from the images WORKDIR.
type fileExtractor struct {
	// Client is the docker client to use to pull the container image.
	client *docker.Client
}

func newFileExtractor(c *docker.Client) *fileExtractor {
	return &fileExtractor{client: c}
}

// Extract implements Extractor Extract.
func (e *fileExtractor) Extract(_ context.Context, img image.Image, w io.Writer) ([]byte, error) {
	c, err := e.createContainer(img)
	if err != nil {
		return nil, err
	}

	defer e.removeContainer(c.ID)

	pfile, err := e.procfile(c.ID)
	if err != nil {
		return nil, err
	}

	b, err := e.copyFile(c.ID, pfile)
	if err != nil {
		return nil, &ProcfileError{Err: err}
	}

	return b, nil
}

// procfile returns the path to the Procfile. If the container has a WORKDIR
// set, then this will return a path to the Procfile within that directory.
func (e *fileExtractor) procfile(id string) (string, error) {
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
func (e *fileExtractor) createContainer(img image.Image) (*docker.Container, error) {
	return e.client.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: img.String(),
		},
	})
}

// removeContainer removes a container by its ID.
func (e *fileExtractor) removeContainer(containerID string) error {
	return e.client.RemoveContainer(docker.RemoveContainerOptions{
		ID: containerID,
	})
}

// copyFile copies a file from a container.
func (e *fileExtractor) copyFile(containerID, path string) ([]byte, error) {
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

func formationFromProcfile(p procfile.Procfile) (Formation, error) {
	switch p := p.(type) {
	case procfile.StandardProcfile:
		return formationFromStandardProcfile(p)
	case procfile.ExtendedProcfile:
		return formationFromExtendedProcfile(p)
	default:
		return nil, &ProcfileError{
			Err: errors.New("unknown Procfile format"),
		}
	}
}

func formationFromStandardProcfile(p procfile.StandardProcfile) (Formation, error) {
	f := make(Formation)

	for name, command := range p {
		cmd, err := ParseCommand(command)
		if err != nil {
			return nil, err
		}

		f[name] = Process{
			Command: cmd,
		}
	}

	return f, nil
}

func formationFromExtendedProcfile(p procfile.ExtendedProcfile) (Formation, error) {
	f := make(Formation)

	for name, process := range p {
		var cmd Command
		var err error

		switch command := process.Command.(type) {
		case string:
			cmd, err = ParseCommand(command)
			if err != nil {
				return nil, err
			}
		case []interface{}:
			for _, v := range command {
				cmd = append(cmd, v.(string))
			}
		default:
			return nil, errors.New("unknown command format")
		}

		f[name] = Process{
			Command: cmd,
		}
	}

	return f, nil
}
