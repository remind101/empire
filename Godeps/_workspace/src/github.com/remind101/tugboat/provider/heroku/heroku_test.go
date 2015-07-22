package heroku

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/remind101/tugboat"
	"golang.org/x/net/context"
)

func TestProvider(t *testing.T) {
	var (
		h *httptest.Server
		g *httptest.Server
	)

	header := `-----> Fetching archive link for ejholmes/acme-inc@abcd... done.
-----> Deploying to acme-inc...
`

	logs := "Hello"

	h = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("%s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/apps/acme-inc/builds":
			fmt.Fprintf(w, `{"id": "1234", "output_stream_url": "%s/stream"}`, h.URL)
		case "/stream":
			fmt.Fprintf(w, logs)
		case "/apps/acme-inc/builds/1234/result":
			fmt.Fprintf(w, `{"build":{"status":"succeeded"}}`)
		}
	}))
	g = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "http://github.com/archive")
		w.WriteHeader(302)
	}))
	defer h.Close()
	defer g.Close()

	p := NewProvider("", "")
	p.heroku.URL = h.URL
	p.github.BaseURL = mustURLParse(g.URL)

	d := &tugboat.Deployment{
		Repo:        "ejholmes/acme-inc",
		Environment: "production",
		Sha:         "abcd",
	}
	w := new(bytes.Buffer)
	if err := p.Deploy(context.Background(), d, w); err != nil {
		t.Fatal(err)
	}

	if got, want := w.String(), header+logs; got != want {
		t.Fatalf("Logs => %s; want %s", got, want)
	}
}

func TestProvider_Failure(t *testing.T) {
	var (
		h *httptest.Server
		g *httptest.Server
	)

	logs := "Hello"

	h = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("%s %s", r.Method, r.URL.Path)

		switch r.URL.Path {
		case "/apps/acme-inc/builds":
			fmt.Fprintf(w, `{"id": "1234", "output_stream_url": "%s/stream"}`, h.URL)
		case "/stream":
			fmt.Fprintf(w, logs)
		case "/apps/acme-inc/builds/1234/result":
			fmt.Fprintf(w, `{"build":{"status":"failed"}}`)
		}
	}))
	g = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "http://github.com/archive")
		w.WriteHeader(302)
	}))
	defer h.Close()
	defer g.Close()

	p := NewProvider("", "")
	p.heroku.URL = h.URL
	p.github.BaseURL = mustURLParse(g.URL)

	d := &tugboat.Deployment{
		Repo:        "ejholmes/acme-inc",
		Environment: "production",
	}
	w := new(bytes.Buffer)
	if err := p.Deploy(context.Background(), d, w); err != tugboat.ErrFailed {
		t.Fatalf("Err => %v; want %v", err, tugboat.ErrFailed)
	}
}

func TestAppFor(t *testing.T) {
	tests := []struct {
		in  tugboat.Deployment
		out string
	}{
		{tugboat.Deployment{Repo: "remind101/r101-api", Environment: "production"}, "r101-api"},
		{tugboat.Deployment{Repo: "remind101/r101-api", Environment: "staging"}, "r101-api-staging"},
	}

	for _, tt := range tests {
		out := appFor(&tt.in)

		if got, want := out, tt.out; got != want {
			t.Errorf("appFor(%v) => %s; want %s", tt.in, got, want)
		}
	}
}

func mustURLParse(uri string) *url.URL {
	u, err := url.Parse(uri)
	if err != nil {
		panic(err)
	}

	return u
}
