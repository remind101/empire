package registry

import (
	"testing"

	"github.com/fsouza/go-dockerclient"
)

func TestGenerator(t *testing.T) {
	tests := []struct {
		auth     *docker.AuthConfigurations
		registry string
		client   *Client
	}{
		{
			auth: &docker.AuthConfigurations{
				Configs: map[string]docker.AuthConfiguration{
					"quay.io": docker.AuthConfiguration{
						Username: "foo",
						Password: "bar",
					},
				},
			},
			registry: "quay.io",
			client: &Client{
				Registry: "quay.io",
				Username: "foo",
				Password: "bar",
			},
		},

		{
			auth: &docker.AuthConfigurations{
				Configs: map[string]docker.AuthConfiguration{
					"https://index.docker.io/v1/": docker.AuthConfiguration{
						Username: "official",
						Password: "password",
					},
					"quay.io": docker.AuthConfiguration{
						Username: "foo",
						Password: "bar",
					},
				},
			},
			registry: "",
			client: &Client{
				Registry: "",
				Username: "official",
				Password: "password",
			},
		},
	}

	for _, tt := range tests {
		g := NewGenerator(tt.auth)

		c, err := g.Generate(tt.registry)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := c.Registry, tt.client.Registry; got != want {
			t.Fatalf("Registry => %s; want %s", got, want)
		}

		if got, want := c.Username, tt.client.Username; got != want {
			t.Fatalf("Username => %s; want %s", got, want)
		}

		if got, want := c.Password, tt.client.Password; got != want {
			t.Fatalf("Password => %s; want %s", got, want)
		}
	}
}
