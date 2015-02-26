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

// NewServer creates the API routes and returns a new Server instance.
func NewServer(e *Empire) *Server {
	r := newRouter()

	// Apps
	r.Handle("GET", "/apps", &GetApps{e.AppsService})                 // hk apps
	r.Handle("DELETE", "/apps/{app}", &DeleteApp{e.AppsService})      // hk destroy
	r.Handle("POST", "/apps", &PostApps{e.AppsService})               // hk create
	r.Handle("POST", "/organizations/apps", &PostApps{e.AppsService}) // hk create

	// Deploys
	r.Handle("POST", "/deploys", &PostDeploys{e.DeploysService}) // Deploy an app

	// Releases
	r.Handle("GET", "/apps/{app}/releases", &GetReleases{e.AppsService, e.ReleasesService}) // hk releases

	// Configs
	r.Handle("GET", "/apps/{app}/config-vars", &GetConfigs{e.AppsService, e.ConfigsService})                        // hk env, hk get
	r.Handle("PATCH", "/apps/{app}/config-vars", &PatchConfigs{e.AppsService, e.ReleasesService, e.ConfigsService}) // hk set

	// Processes
	r.Handle("GET", "/apps/{app}/dynos", &GetProcesses{e.AppsService, e.Manager}) // hk dynos

	// Formations
	r.Handle("PATCH", "/apps/{app}/formation", &PatchFormation{e.AppsService, e.ReleasesService, e.Manager}) // hk scale

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

type DeleteApp struct {
	AppsService AppsService
}

func (h *DeleteApp) Serve(req *Request) (int, interface{}, error) {
	name := AppName(req.Vars["app"])

	a, err := h.AppsService.FindByName(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	if err := h.AppsService.Destroy(a); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 200, nil, nil
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

	d, err := h.DeploysService.Deploy(Image{
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

type GetConfigs struct {
	AppsService    AppsService
	ConfigsService ConfigsService
}

func (h *GetConfigs) Serve(req *Request) (int, interface{}, error) {
	name := AppName(req.Vars["app"])

	a, err := h.AppsService.FindByName(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	c, err := h.ConfigsService.Head(a)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 200, c.Vars, nil
}

type PatchConfigs struct {
	AppsService     AppsService
	ReleasesService ReleasesService
	ConfigsService  ConfigsService
}

func (h *PatchConfigs) Serve(req *Request) (int, interface{}, error) {
	var configVars Vars

	if err := req.Decode(&configVars); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	name := AppName(req.Vars["app"])

	// Find app
	a, err := h.AppsService.FindByName(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	// Update the config
	c, err := h.ConfigsService.Apply(a, configVars)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	// Find current release
	r, err := h.ReleasesService.Head(a)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	// If there is an existing release, create a new one
	if r != nil {
		// Create new release based on new config and old slug
		_, err = h.ReleasesService.Create(a, c, r.Slug)
		if err != nil {
			return http.StatusInternalServerError, nil, err
		}
	}

	return 200, c.Vars, nil
}

type GetProcesses struct {
	AppsService AppsService
	Manager     Manager
}

type dyno struct {
	Command string `json:"command"`
	Name    string `json:"name"`
	State   string `json:"state"`
}

func (h *GetProcesses) Serve(req *Request) (int, interface{}, error) {
	name := AppName(req.Vars["app"])

	a, err := h.AppsService.FindByName(name)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	if a == nil {
		return http.StatusNotFound, nil, nil
	}

	// Retrieve job states
	js, err := h.Manager.JobStatesByApp(a)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	// Convert to hk compatible format
	dynos := make([]dyno, len(js))
	for i, j := range js {
		dynos[i] = dyno{
			Command: string(j.Job.Command),
			Name:    string(j.Name),
			State:   j.State,
		}
	}

	return 200, dynos, nil
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

	if r == nil {
		return http.StatusNotFound, nil, nil
	}

	err = h.Manager.ScaleRelease(r, qm)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 200, nil, nil
}
