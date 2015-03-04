package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/remind101/empire/empire"
)

// Decoder represents a function that can decode a request into an interface
// value.
type Decoder func(r *http.Request, v interface{}) error

func JSONDecode(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// Request wraps an http.Request for convenience.
type Request struct {
	*http.Request
	Vars map[string]string
	Decoder
}

// NewRequest parse the mux vars and returns a new Request instance.
func NewRequest(r *http.Request) *Request {
	return &Request{Request: r, Vars: mux.Vars(r)}
}

// Decode decodes the request using the Decoder.
func (r *Request) Decode(v interface{}) error {
	d := r.Decoder

	if d == nil {
		d = JSONDecode
	}

	return d(r.Request, v)
}

// Handler defines an interface for service an HTTP request.
type Handler interface {
	Serve(*Request) (int, interface{}, error)
}

// ErrorResource represents the error response format that we return.
type ErrorResource struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	URL     string `json:"url"`
}

// Error implements error interface.
func (e *ErrorResource) Error() string {
	return e.Message
}

// Endpoint wraps a Handler to implement the http.Handler interface.
type Endpoint struct {
	Handler
}

// ServeHTTP implements the http.Handler interface. It will parse the form
// params, then serve the request, finally JSON encoding the returned value.
func (e *Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := NewRequest(r)

	status, v, err := e.Handler.Serve(req)
	if err != nil {
		if _, ok := err.(*ErrorResource); ok {
			v = err
		} else {
			v = &ErrorResource{Message: err.Error()}
		}
		log.Printf("Error: %v\n", v)
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Server represents the API.
type Server struct {
	http.Handler
}

// New creates the API routes and returns a new Server instance.
func New(e *empire.Empire) *Server {
	r := newRouter()

	// Apps
	r.Handle("GET", "/apps", &GetApps{e})                 // hk apps
	r.Handle("DELETE", "/apps/{app}", &DeleteApp{e})      // hk destroy
	r.Handle("POST", "/apps", &PostApps{e})               // hk create
	r.Handle("POST", "/organizations/apps", &PostApps{e}) // hk create

	// Deploys
	r.Handle("POST", "/deploys", &PostDeploys{e.DeploysService}) // Deploy an app

	// Releases
	r.Handle("GET", "/apps/{app}/releases", &GetReleases{e, e.ReleasesService})                                     // hk releases
	r.Handle("POST", "/apps/{app}/releases", &PostReleases{e, e.ReleasesService, e.ConfigsService, e.SlugsService}) // hk rollback

	// Configs
	r.Handle("GET", "/apps/{app}/config-vars", &GetConfigs{e, e.ConfigsService})                                        // hk env, hk get
	r.Handle("PATCH", "/apps/{app}/config-vars", &PatchConfigs{e, e.ReleasesService, e.ConfigsService, e.SlugsService}) // hk set

	// Processes
	r.Handle("GET", "/apps/{app}/dynos", &GetProcesses{e, e.JobsService}) // hk dynos

	// Formations
	r.Handle("PATCH", "/apps/{app}/formation", &PatchFormation{e, e.ReleasesService, e.ConfigsService, e.SlugsService, e.ProcessesService, e.Manager}) // hk scale

	n := negroni.Classic()
	n.UseHandler(r)

	return &Server{n}
}

// router is an http router for Handlers.
type router struct {
	*mux.Router
}

// newRouter returns a new router instance.
func newRouter() *router {
	return &router{Router: mux.NewRouter()}
}

// Handle sets up a route for a Handler.
func (r *router) Handle(method, path string, h Handler) {
	r.Router.Handle(path, &Endpoint{Handler: h}).Methods(method)
}
