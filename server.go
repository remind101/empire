package empire

import (
	"encoding/json"
	"net/http"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/deploys"
	"github.com/remind101/empire/repos"
	"github.com/remind101/empire/slugs"
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
	Err string `json:"error"`
}

// Error implements error interface.
func (e *ErrorResource) Error() string {
	return e.Err
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
			v = &ErrorResource{Err: err.Error()}
		}
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Server represents the API.
type Server struct {
	http.Handler
}

// NewServer creates the API routes and returns a new Server instance.
func NewServer(e *Empire) *Server {
	r := newRouter()

	r.Handle("POST", "/deploys", &PostDeploys{e.DeploysService()})
	r.Handle("POST", "/apps", &PostApps{e.AppsService()})
	r.Handle("PATCH", "/apps/{app}/configs", &PostConfigs{e.AppsService(), e.ConfigsService()})

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

// PostDeploys is a Handler for the POST /v1/deploys endpoint.
type PostDeploys struct {
	DeploysService *deploys.Service
}

// PostDeployForm is the form object that represents the POST body.
type PostDeployForm struct {
	Image struct {
		ID   string `json:"id"`
		Repo string `json:"repo"`
	} `json:"image"`
}

// Serve implements the Handler interface.
func (h *PostDeploys) Serve(req *Request) (int, interface{}, error) {
	var form PostDeployForm

	if err := req.Decode(&form); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	d, err := h.DeploysService.Deploy(&slugs.Image{
		Repo: repos.Repo(form.Image.Repo),
		ID:   form.Image.ID,
	})
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 201, d, nil
}

type PostAppsForm struct {
	Repo string `json:"repo"`
}

type PostApps struct {
	AppsService *apps.Service
}

func (h *PostApps) Serve(req *Request) (int, interface{}, error) {
	var form PostAppsForm

	if err := req.Decode(&form); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	a, err := h.AppsService.Create(&apps.App{
		Repo: repos.Repo(form.Repo),
	})
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 201, a, nil
}

type PostConfigs struct {
	AppsService    *apps.Service
	ConfigsService *configs.Service
}

type PostConfigsForm struct {
	Vars configs.Vars `json:"vars"`
}

func (h *PostConfigs) Serve(req *Request) (int, interface{}, error) {
	var form PostConfigsForm

	if err := req.Decode(&form); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	id := apps.ID(req.Vars["app"])

	a, err := h.AppsService.FindByID(id)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	c, err := h.ConfigsService.Apply(a, form.Vars)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 200, c, nil
}
