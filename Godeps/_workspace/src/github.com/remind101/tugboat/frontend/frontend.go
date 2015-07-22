package frontend

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"text/template"

	"github.com/codegangsta/negroni"
)

var (
	// DefaultDirectory is the path to the compiled frontend.
	DefaultDirectory = "frontend/dist"

	// DefaultIndex is the name of the fallback file to render.
	DefaultIndex = "/index.html"
)

// Handler is an http.Handler that serves the frontend.
type Handler struct {
	PusherKey string

	Dir    string
	Index  string
	static *negroni.Static

	index []byte
}

// New returns a new Handler instance.
func New(dir string) *Handler {
	if dir == "" {
		dir = DefaultDirectory
	}

	s := negroni.NewStatic(http.Dir(dir))
	s.IndexFile = ""
	return &Handler{
		Dir:    dir,
		Index:  DefaultIndex,
		static: s,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.static.ServeHTTP(w, r, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(w, h.indexReader())
	}))
}

func (h *Handler) indexReader() io.Reader {
	if h.index != nil {
		return bytes.NewReader(h.index)
	}

	raw, err := ioutil.ReadFile(h.Dir + h.Index)
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("index").Parse(string(raw))
	if err != nil {
		panic(err)
	}

	data := struct {
		PusherKey string
	}{
		PusherKey: h.PusherKey,
	}

	b := new(bytes.Buffer)
	if err := tmpl.Execute(b, data); err != nil {
		panic(err)
	}

	h.index = b.Bytes()

	return bytes.NewReader(h.index)
}
