package empire

import (
	"archive/tar"
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/fsouza/go-dockerclient"
)

func TestFakeExtractor(t *testing.T) {
	e := fakeExtractor{}

	got, err := e.Extract(Image{})
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
	api := newReplayHandler(t).Add(testPathHandler(t,
		"GET /images/remind101:acme-inc/json",
		200, `{ "Config": { "Cmd": ["/go/bin/app","server"] } }`,
	))

	c, s := newTestDockerClient(t, api)
	defer s.Close()

	e := cmdExtractor{
		client: c,
	}

	got, err := e.Extract(Image{
		ID:   "acme-inc",
		Repo: "remind101",
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
	api := newReplayHandler(t).Add(testPathHandler(t,
		"POST /containers/create",
		200, `{ "ID": "abc" }`,
	)).Add(testPathHandler(t,
		"GET /containers/abc/json",
		200, `{}`,
	)).Add(testPathHandler(t,
		"POST /containers/abc/copy",
		200, tarProcfile(t),
	)).Add(testPathHandler(t,
		"DELETE /containers/abc",
		200, `{}`,
	))

	c, s := newTestDockerClient(t, api)
	defer s.Close()

	e := procfileExtractor{
		client: c,
	}

	got, err := e.Extract(Image{
		ID:   "acme-inc",
		Repo: "remind101",
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
	api := newReplayHandler(t).Add(testPathHandler(t,
		"POST /containers/create",
		200, `{ "ID": "abc" }`,
	)).Add(testPathHandler(t,
		"GET /containers/abc/json",
		200, `{}`,
	)).Add(testPathHandler(t,
		"POST /containers/abc/copy",
		404, ``,
	)).Add(testPathHandler(t,
		"DELETE /containers/abc",
		200, `{}`,
	)).Add(testPathHandler(t,
		"GET /images/remind101:acme-inc/json",
		200, `{ "Config": { "Cmd": ["/go/bin/app","server"] } }`,
	))

	c, s := newTestDockerClient(t, api)
	defer s.Close()

	e := newProcfileFallbackExtractor(c)

	got, err := e.Extract(Image{
		ID:   "acme-inc",
		Repo: "remind101",
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

type replayHandler struct {
	t        *testing.T
	i        int
	handlers []http.Handler
}

func newReplayHandler(t *testing.T) *replayHandler {
	return &replayHandler{t: t, handlers: make([]http.Handler, 0)}
}

func (h *replayHandler) Add(handler http.Handler) *replayHandler {
	h.handlers = append(h.handlers, handler)
	return h
}

func (h *replayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.i >= len(h.handlers) {
		h.t.Errorf("http request: %s %s; no more handlers to replay", r.Method, r.URL.Path)
	} else {
		h.handlers[h.i].ServeHTTP(w, r)
		h.i++
	}
}

func testPathHandler(t *testing.T, methPath string, respStatus int, respBody string) http.Handler {
	s := strings.SplitN(methPath, " ", 2)
	method := s[0]
	path := s[1]

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if method == r.Method && path == r.URL.Path {
			w.WriteHeader(respStatus)
			w.Write([]byte(respBody))
		} else {
			w.WriteHeader(http.StatusNotFound)
			t.Errorf("http request => %s %s; want %s %s", r.Method, r.URL.Path, method, path)
		}
	})
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
