package extractor

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"path"

	"golang.org/x/net/context"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/procfile"
)

// PullAndExtract returns a ProcfileExtractor that will pull the image using the
// docker client, then attempt to extract the the Procfile, or fallback to the
// CMD directive in the Dockerfile.
func PullAndExtract(c *dockerutil.Client) empire.ProcfileExtractor {
	e := multiExtractor(
		newFileExtractor(c),
		newCMDExtractor(c),
	)

	return empire.ProcfileExtractorFunc(func(ctx context.Context, img image.Image, w io.Writer) ([]byte, error) {
		if err := c.PullImage(ctx, docker.PullImageOptions{
			Registry:      img.Registry,
			Repository:    img.Repository,
			Tag:           img.Tag,
			OutputStream:  w,
			RawJSONStream: true,
		}); err != nil {
			return nil, err
		}

		return e.Extract(ctx, img, w)
	})
}

// cmdExtractor is an Extractor implementation that returns a Procfile based
// on the CMD directive in the Dockerfile. It makes the assumption that the cmd
// is a "web" process.
type cmdExtractor struct {
	// Client is the docker client to use to pull the container image.
	client *dockerutil.Client
}

func newCMDExtractor(c *dockerutil.Client) *cmdExtractor {
	return &cmdExtractor{client: c}
}

func (e *cmdExtractor) Extract(ctx context.Context, img image.Image, _ io.Writer) ([]byte, error) {
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
func multiExtractor(extractors ...empire.ProcfileExtractor) empire.ProcfileExtractor {
	return empire.ProcfileExtractorFunc(func(ctx context.Context, image image.Image, w io.Writer) ([]byte, error) {
		for _, extractor := range extractors {
			p, err := extractor.Extract(ctx, image, w)

			// Yay!
			if err == nil {
				return p, nil
			}

			// Try the next one
			if _, ok := err.(*empire.ProcfileError); ok {
				continue
			}

			// Bubble up the error
			return p, err
		}

		return nil, &empire.ProcfileError{
			Err: errors.New("no suitable Procfile extractor found"),
		}
	})
}

// fileExtractor is an implementation of the Extractor interface that extracts
// the Procfile from the images WORKDIR.
type fileExtractor struct {
	// Client is the docker client to use to pull the container image.
	client *dockerutil.Client
}

func newFileExtractor(c *dockerutil.Client) *fileExtractor {
	return &fileExtractor{client: c}
}

// Extract implements Extractor Extract.
func (e *fileExtractor) Extract(ctx context.Context, img image.Image, w io.Writer) ([]byte, error) {
	c, err := e.createContainer(ctx, img)
	if err != nil {
		return nil, err
	}

	defer e.removeContainer(ctx, c.ID)

	pfile, err := e.procfile(ctx, c.ID)
	if err != nil {
		return nil, err
	}

	b, err := e.copyFile(ctx, c.ID, pfile)
	if err != nil {
		return nil, &empire.ProcfileError{Err: err}
	}

	return b, nil
}

// procfile returns the path to the Procfile. If the container has a WORKDIR
// set, then this will return a path to the Procfile within that directory.
func (e *fileExtractor) procfile(ctx context.Context, id string) (string, error) {
	p := ""

	c, err := e.client.InspectContainer(id)
	if err != nil {
		return "", err
	}

	if c.Config != nil {
		p = c.Config.WorkingDir
	}

	return path.Join(p, empire.Procfile), nil
}

// createContainer creates a new docker container for the given docker image.
func (e *fileExtractor) createContainer(ctx context.Context, img image.Image) (*docker.Container, error) {
	return e.client.CreateContainer(ctx, docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: img.String(),
		},
	})
}

// removeContainer removes a container by its ID.
func (e *fileExtractor) removeContainer(ctx context.Context, containerID string) error {
	return e.client.RemoveContainer(ctx, docker.RemoveContainerOptions{
		ID: containerID,
	})
}

// copyFile copies a file from a container.
func (e *fileExtractor) copyFile(ctx context.Context, containerID, path string) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.client.CopyFromContainer(ctx, docker.CopyFromContainerOptions{
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
