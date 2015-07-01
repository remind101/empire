package empire

import (
	"archive/tar"
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/httpmock"
	"github.com/remind101/empire/pkg/image"
)

func TestFakeExtractor(t *testing.T) {
	e := fakeExtractor{}

	got, err := e.Extract(image.Image{})
	if err != nil {
		t.Fatal(err)
	}

	want := CommandMap{
		ProcessType("web"): Command("./bin/web"),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Extract() => %q; want %q", got, want)
	}
}

func TestCmdExtractor(t *testing.T) {
	api := httpmock.NewServeReplay(t).Add(httpmock.PathHandler(t,
		"GET /images/remind101:acme-inc/json",
		200, `{ "Config": { "Cmd": ["/go/bin/app","server"] } }`,
	))

	c, s := newTestDockerClient(t, api)
	defer s.Close()

	e := cmdExtractor{
		client: c,
	}

	got, err := e.Extract(image.Image{
		Tag:        "acme-inc",
		Repository: "remind101",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := CommandMap{
		ProcessType("web"): Command("/go/bin/app server"),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Extract() => %q; want %q", got, want)
	}
}

func TestProcfileExtractor(t *testing.T) {
	api := httpmock.NewServeReplay(t).Add(httpmock.PathHandler(t,
		"POST /containers/create",
		200, `{ "ID": "abc" }`,
	)).Add(httpmock.PathHandler(t,
		"GET /containers/abc/json",
		200, `{}`,
	)).Add(httpmock.PathHandler(t,
		"POST /containers/abc/copy",
		200, tarProcfile(t),
	)).Add(httpmock.PathHandler(t,
		"DELETE /containers/abc",
		200, `{}`,
	))

	c, s := newTestDockerClient(t, api)
	defer s.Close()

	e := procfileExtractor{
		client: c,
	}

	got, err := e.Extract(image.Image{
		Tag:        "acme-inc",
		Repository: "remind101",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := CommandMap{
		ProcessType("web"): Command("rails server"),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Extract() => %q; want %q", got, want)
	}

}

func TestProcfileFallbackExtractor(t *testing.T) {
	api := httpmock.NewServeReplay(t).Add(httpmock.PathHandler(t,
		"POST /containers/create",
		200, `{ "ID": "abc" }`,
	)).Add(httpmock.PathHandler(t,
		"GET /containers/abc/json",
		200, `{}`,
	)).Add(httpmock.PathHandler(t,
		"POST /containers/abc/copy",
		404, ``,
	)).Add(httpmock.PathHandler(t,
		"DELETE /containers/abc",
		200, `{}`,
	)).Add(httpmock.PathHandler(t,
		"GET /images/remind101:acme-inc/json",
		200, `{ "Config": { "Cmd": ["/go/bin/app","server"] } }`,
	))

	c, s := newTestDockerClient(t, api)
	defer s.Close()

	e := newProcfileFallbackExtractor(c)

	got, err := e.Extract(image.Image{
		Tag:        "acme-inc",
		Repository: "remind101",
	})
	if err != nil {
		t.Fatal(err)
	}

	want := CommandMap{
		ProcessType("web"): Command("/go/bin/app server"),
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Extract() => %q; want %q", got, want)
	}

}

// newTestDockerClient returns a docker.Client configured to talk to the given http.Handler
func newTestDockerClient(t *testing.T, fakeDockerAPI http.Handler) (*docker.Client, *httptest.Server) {
	s := httptest.NewServer(fakeDockerAPI)

	c, err := docker.NewClient(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	return c, s
}

func tarProcfile(t *testing.T) string {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	var files = []struct {
		Name, Body string
	}{
		{"Procfile", "web: rails server"},
	}

	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Size: int64(len(file.Body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(file.Body)); err != nil {
			t.Fatal(err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.String()
}
