package dockerauth

import (
	"io"

	"github.com/fsouza/go-dockerclient"
)

type dockerConfigAuthProvider struct {
	authConfs *docker.AuthConfigurations
}

func NewDockerConfigAuthProvider(file io.Reader) (*dockerConfigAuthProvider, error) {
	authConfs, err := docker.NewAuthConfigurations(file)
	if err != nil {
		return nil, err
	}

	return &dockerConfigAuthProvider{
		authConfs: authConfs,
	}, nil
}

func (p *dockerConfigAuthProvider) AuthConfiguration(registry string) (*docker.AuthConfiguration, error) {
	if registry == "" {
		registry = "https://index.docker.io/v1/"
	}

	authConf, exists := p.authConfs.Configs[registry]

	if !exists {
		return nil, nil
	}

	return &authConf, nil
}
