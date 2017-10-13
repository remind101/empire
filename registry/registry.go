package registry

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"

	"golang.org/x/net/context"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/pkg/jsonmessage"
	"github.com/remind101/empire/procfile"
)

type Digest int

const (
	// Uses the digest, if it's available. Older versions of Docker do not
	// always populate the digests.
	//
	// See https://github.com/moby/moby/issues/15508
	DigestsPrefer Digest = iota

	// Always use the digest, and return an error if the image doesn't have
	// one. The most secure option (and prefered).
	DigestsOnly

	// Disabling digests entirely, and just use the mutable tag. Unsecure,
	// and not recommended.
	DigestsDisable
)

// NoDigestError can be returned if image digests are enforce, and the image
// doesn't have an immutable digest.
type NoDigestError struct {
	Image string
}

// Error implements the error interface.
func (e *NoDigestError) Error() string {
	return fmt.Sprintf("image %s has no digests", e.Image)
}

// DockerDaemonRegistry is an implementation of the empire.ImageRegistry interface
// backed by a local Docker daemon.
type DockerDaemonRegistry struct {
	// Controls whether digests are preferred, enforced, or disabled. See
	// above for available options.
	Digests Digest

	docker    *dockerutil.Client
	extractor empire.ProcfileExtractor

	// can be set to false to disable Pull-before-Resolve.
	noPull bool
}

// DockerDaemon returns an empire.ImageRegistry that uses a local Docker Daemon
// to extract procfiles and resolve images.
func DockerDaemon(c *dockerutil.Client) *DockerDaemonRegistry {
	e := multiExtractor(
		newFileExtractor(c),
		newCMDExtractor(c),
	)
	return &DockerDaemonRegistry{
		docker:    c,
		extractor: e,
	}
}

func (r *DockerDaemonRegistry) ExtractProcfile(ctx context.Context, img image.Image, w *jsonmessage.Stream) ([]byte, error) {
	return r.extractor.ExtractProcfile(ctx, img, w)
}

func (r *DockerDaemonRegistry) Resolve(ctx context.Context, img image.Image, w *jsonmessage.Stream) (image.Image, error) {
	if !r.noPull {
		pullOptions, err := dockerutil.PullImageOptions(img)
		if err != nil {
			return img, err
		}

		pullOptions.OutputStream = w
		pullOptions.RawJSONStream = true

		if err := r.docker.PullImage(ctx, pullOptions); err != nil {
			return img, err
		}
	}

	// If digests are disable, just return the original image reference.
	if r.Digests == DigestsDisable {
		return img, nil
	}

	// If the image already references an immutable identifier, there's
	// nothing for us to do.
	if img.Digest != "" {
		return img, nil
	}

	i, err := r.docker.InspectImage(img.String())
	if err != nil {
		return img, err
	}

	// If there are no repository digests (the case for Docker <= 1.11),
	// then we just fallback to the original identifier.
	if len(i.RepoDigests) <= 0 {
		switch r.Digests {
		case DigestsPrefer:
			w.Encode(jsonmessage.JSONMessage{
				Status: fmt.Sprintf("Status: Image has no repository digests. Using %s as image identifier", img),
			})
			return img, nil
		case DigestsOnly:
			return img, &NoDigestError{Image: img.String()}
		}
	}

	digest := i.RepoDigests[0]

	w.Encode(jsonmessage.JSONMessage{
		Status: fmt.Sprintf("Status: Resolved %s to %s", img, digest),
	})

	return image.Decode(digest)
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

func (e *cmdExtractor) ExtractProcfile(ctx context.Context, img image.Image, w *jsonmessage.Stream) ([]byte, error) {
	i, err := e.client.InspectImage(img.String())
	if err != nil {
		return nil, err
	}

	w.Encode(jsonmessage.JSONMessage{
		Status: fmt.Sprintf("Status: Generating Procfile from CMD: %v", i.Config.Cmd),
	})

	return procfile.Marshal(procfile.ExtendedProcfile{
		"web": procfile.Process{
			Command: i.Config.Cmd,
		},
	})
}

// multiExtractor is an Extractor implementation that tries multiple Extractors
// in succession until one succeeds.
func multiExtractor(extractors ...empire.ProcfileExtractor) empire.ProcfileExtractor {
	return empire.ProcfileExtractorFunc(func(ctx context.Context, image image.Image, w *jsonmessage.Stream) ([]byte, error) {
		for _, extractor := range extractors {
			p, err := extractor.ExtractProcfile(ctx, image, w)

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
func (e *fileExtractor) ExtractProcfile(ctx context.Context, img image.Image, w *jsonmessage.Stream) ([]byte, error) {
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

	w.Encode(jsonmessage.JSONMessage{
		Status: fmt.Sprintf("Status: Extracted Procfile from %q", pfile),
	})

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
