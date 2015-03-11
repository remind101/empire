package registry

import (
	"io"
	"net/http"
	"testing"
)

func TestMultiClientResolveTag(t *testing.T) {
	c, s := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `"1234"`)
	}))
	defer s.Close()

	r := &MultiClient{
		generator: GeneratorFunc(func(registry string) (*Client, error) {
			if got, want := registry, "quay.io"; got != want {
				t.Fatalf("Registry => %s; want %s", got, want)
			}

			return c, nil
		}),
	}

	imageID, err := r.ResolveTag("quay.io/ejholmes/acme-inc", "abc21")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := imageID, "1234"; got != want {
		t.Fatal("ImageID => %s; want %s", got, want)
	}
}

type GeneratorFunc func(string) (*Client, error)

func (f GeneratorFunc) Generate(registry string) (*Client, error) {
	return f(registry)
}
