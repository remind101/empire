package dockerauth

import "github.com/fsouza/go-dockerclient"

type multiAuthProvider struct {
	providers []AuthProvider
}

func NewMultiAuthProvider(providers ...AuthProvider) *multiAuthProvider {
	return &multiAuthProvider{
		providers: providers,
	}
}

func (p *multiAuthProvider) AuthConfiguration(registry string) (*docker.AuthConfiguration, error) {
	for _, provider := range p.providers {
		authConf, err := provider.AuthConfiguration(registry)
		if err != nil {
			return nil, err
		}

		if authConf != nil {
			return authConf, nil
		}
	}

	return nil, nil
}

func (p *multiAuthProvider) AddProvider(provider AuthProvider) {
	p.providers = append(p.providers, provider)
}
