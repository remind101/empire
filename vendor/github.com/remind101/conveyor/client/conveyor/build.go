package conveyor

import (
	"fmt"
	"io"
	"time"
)

// Build is a helper around the BuildCreate, BuildInfo, LogsStream and
// ArtifactInfo methods to ultimately return an Artifact and stream any
// build logs.
func (s *Service) Build(w io.Writer, o BuildCreateOpts) (*Artifact, error) {
	repoSha := fmt.Sprintf("%s@%s", o.Repository, o.Sha)

	a, err := s.ArtifactInfo(repoSha)
	if err == nil {
		return a, nil
	}

	if !notFound(err) {
		return nil, err
	}

	b, err := s.BuildInfo(repoSha)
	if err != nil {
		if !notFound(err) {
			return nil, err
		}

		// No build, create one
		b, err = s.BuildCreate(o)
		if err != nil {
			return nil, err
		}
	}

	buildID := b.ID

	// TODO: Stream the logs.

	for {
		<-time.After(5 * time.Second)

		b, err = s.BuildInfo(buildID)
		if err != nil {
			return nil, err
		}

		// If the build failed, return an error.
		if b.State == "failed" {
			return nil, fmt.Errorf("build %s failed", buildID)
		}

		// If the build completed, we should have an artifact.
		if b.CompletedAt != nil {
			break
		}
	}

	return s.ArtifactInfo(repoSha)
}
