package registry

import (
	"fmt"

	"github.com/fsouza/go-dockerclient"
)

// Generator is a Client generator. Initialize this with a set of
// docker.AuthConfigurations, then call the Generate method to generate a new
// authenticated Client for the given registry.
type Generator struct {
	Auth *docker.AuthConfigurations
}

func NewGenerator(auth *docker.AuthConfigurations) *Generator {
	return &Generator{Auth: auth}
}

// Generate returns a new Client instance if an AuthConfiguration is found for
// it.
func (g *Generator) Generate(registry string) (*Client, error) {
	reg := registry

	if reg == "" {
		reg = fmt.Sprintf("https://%s/v1/", DefaultRegistry)
	}

	if g.Auth == nil {
		return nil, AuthError{Registry: reg}
	}

	auth, ok := g.Auth.Configs[reg]
	if !ok {
		return nil, AuthError{Registry: reg}
	}

	c := NewClient(nil)
	c.Registry = registry
	c.Username = auth.Username
	c.Password = auth.Password

	return c, nil
}

// AuthError can be returned by the generator if an AuthConfiguration can't be
// found for the registry.
type AuthError struct {
	// The registry that this pertains to.
	Registry string
}

// Error implements the error interface.
func (e AuthError) Error() string {
	return fmt.Sprintf("registry: no auth configuration: %s", e.Registry)
}
