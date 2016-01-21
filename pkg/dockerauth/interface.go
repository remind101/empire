package dockerauth

import "github.com/fsouza/go-dockerclient"

type AuthProvider interface {
	AuthConfiguration(registry string) (*docker.AuthConfiguration, error)
}
