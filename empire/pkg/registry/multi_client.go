package registry

import "github.com/fsouza/go-dockerclient"

// MultiClient implements the same interface as Client, but is backed by a
// Generator to generate a Client depending on the docker repo.
type MultiClient struct {
	generator interface {
		Generate(registry string) (*Client, error)
	}
}

// NewMultiClient returns a new MultiClient instance with a generator backed by
// the docker auth configurations.
func NewMultiClient(auth *docker.AuthConfigurations) *MultiClient {
	return &MultiClient{
		generator: NewGenerator(auth),
	}
}

// Resolve resolves the given tag to an image id within the repo.
func (mc *MultiClient) ResolveTag(fullRepo, tag string) (string, error) {
	c, repo, err := mc.generateClient(fullRepo)
	if err != nil {
		return "", err
	}

	return c.ResolveTag(repo, tag)
}

func (mc *MultiClient) generateClient(fullRepo string) (*Client, string, error) {
	registry, repo, err := Split(fullRepo)
	if err != nil {
		return nil, repo, err
	}

	c, err := mc.generator.Generate(registry)
	if err != nil {
		return nil, repo, err
	}

	return c, repo, nil
}
