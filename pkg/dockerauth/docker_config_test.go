package dockerauth

import (
	"testing"

	"github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestDockerConfigAuthProvider_AuthConfiguration(t *testing.T) {
	provider := dockerConfigAuthProvider{
		authConfs: &docker.AuthConfigurations{
			Configs: map[string]docker.AuthConfiguration{
				"foobar": docker.AuthConfiguration{
					Username: "foo",
				},
				"https://index.docker.io/v1/": docker.AuthConfiguration{
					Username: "hubuser",
				},
			},
		},
	}

	authConf, err := provider.AuthConfiguration("foobar")
	assert.NoError(t, err)
	assert.Equal(t, "foo", authConf.Username)

	authConf, err = provider.AuthConfiguration("foobaz")
	assert.NoError(t, err)
	assert.Nil(t, authConf)

	authConf, err = provider.AuthConfiguration("")
	assert.NoError(t, err)
	assert.Equal(t, "hubuser", authConf.Username)
}
