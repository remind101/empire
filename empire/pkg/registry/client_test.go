package registry

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestClientResolveTag(t *testing.T) {
	c, s := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `"1234"`)
	}))
	defer s.Close()

	imageID, err := c.ResolveTag("repo", "commit")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := imageID, "1234"; got != want {
		t.Fatal("Image ID => %s; want %s", got, want)
	}
}

func newTestClient(t testing.TB, h http.Handler) (*Client, *httptest.Server) {
	s := httptest.NewServer(h)

	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	c := NewClient(nil)
	c.Registry = u.Host
	c.DisableTLS = true

	return c, s
}
