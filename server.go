package empire

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
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
		log.Printf("Error: %v\n", v)
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

	// Apps
	r.Handle("GET", "/apps", &GetApps{e.AppsService})   // List existing apps
	r.Handle("POST", "/apps", &PostApps{e.AppsService}) // Create a new app

	// Deploys
	r.Handle("POST", "/deploys", &PostDeploys{e.DeploysService}) // Deploy an app

	// Releases
	r.Handle("GET", "/apps/{app}/releases", &GetReleases{e.AppsService, e.ReleasesService}) // List existing releases

	// Configs
	r.Handle("PATCH", "/apps/{app}/configs", &PatchConfigs{e.AppsService, e.ConfigsService}) // Update an app config

	// Formations
	r.Handle("PATCH", "/apps/{app}/formation", &PatchFormation{e.AppsService, e.ReleasesService, e.Manager}) // Batch update formation

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

type GetApps struct {
	AppsService AppsService
}

func (h *GetApps) Serve(req *Request) (int, interface{}, error) {
	apps, err := h.AppsService.FindAll()
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 200, apps, nil
}

type PostAppsForm struct {
	Name string `json:"name"`
	Repo string `json:"repo"`
}

type PostApps struct {
	AppsService AppsService
}

func (h *PostApps) Serve(req *Request) (int, interface{}, error) {
	var form PostAppsForm

	if err := req.Decode(&form); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	app, err := NewApp(AppName(form.Name), Repo(form.Repo))
	if err != nil {
		return http.StatusBadRequest, nil, err
	}

	a, err := h.AppsService.Create(app)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 201, a, nil
}

// PostDeploys is a Handler for the POST /v1/deploys endpoint.
type PostDeploys struct {
	DeploysService DeploysService
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

	d, err := h.DeploysService.Deploy(&Image{
		Repo: Repo(form.Image.Repo),
		ID:   form.Image.ID,
	})
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 201, d, nil
}

type GetReleases struct {
	AppsService     AppsService
	ReleasesService ReleasesService
}

func (h *GetReleases) Serve(req *Request) (int, interface{}, error) {
	name := AppName(req.Vars["app"])

	a, err := h.AppsService.FindByName(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	rels, err := h.ReleasesService.FindByApp(a)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 200, rels, nil
}

type PatchConfigs struct {
	AppsService    AppsService
	ConfigsService ConfigsService
}

type PatchConfigsForm struct {
	Vars Vars `json:"vars"`
}

func (h *PatchConfigs) Serve(req *Request) (int, interface{}, error) {
	var form PatchConfigsForm

	if err := req.Decode(&form); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	name := AppName(req.Vars["app"])

	a, err := h.AppsService.FindByName(name)
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

type PatchFormation struct {
	AppsService     AppsService
	ReleasesService ReleasesService
	Manager         Manager
}

type PatchFormationForm struct {
	Updates []struct {
		Process  string `json:"process"` // Refers to process type
		Quantity int    `json:"quantity"`
		Size     string `json:"size"`
	} `json:"updates"`
}

func (h *PatchFormation) Serve(req *Request) (int, interface{}, error) {
	var form PatchFormationForm

	if err := req.Decode(&form); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	name := AppName(req.Vars["app"])

	a, err := h.AppsService.FindByName(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	qm := ProcessQuantityMap{}
	for _, up := range form.Updates {
		qm[ProcessType(up.Process)] = up.Quantity
	}

	r, err := h.ReleasesService.Head(a)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	err = h.Manager.ScaleRelease(r, qm)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 200, nil, nil
}
