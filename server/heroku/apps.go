package heroku

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/heroku"
	"github.com/remind101/pkg/reporter"
)

type App heroku.App

func newApp(a *empire.App) *App {
	return &App{
		Id:        a.ID,
		Name:      a.Name,
		CreatedAt: *a.CreatedAt,
		Cert:      a.Cert,
	}
}

func newApps(as []*empire.App) []*App {
	apps := make([]*App, len(as))

	for i := 0; i < len(as); i++ {
		apps[i] = newApp(as[i])
	}

	return apps
}

func (h *Server) GetApps(w http.ResponseWriter, r *http.Request) error {
	apps, err := h.Apps(empire.AppsQuery{})
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newApps(apps))
}

func (h *Server) GetAppInfo(w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(r, h)
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newApp(a))
}

func (h *Server) DeleteApp(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	a, err := findApp(r, h)
	if err != nil {
		return err
	}

	m, err := findMessage(r)
	if err != nil {
		return err
	}

	if err := h.Destroy(ctx, empire.DestroyOpts{
		User:    UserFromContext(ctx),
		App:     a,
		Message: m,
	}); err != nil {
		return err
	}

	return NoContent(w)
}

func (h *Server) DeployApp(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	a, err := findApp(r, h)
	if err != nil {
		return err
	}

	opts, err := newDeployOpts(ctx, w, r)
	opts.App = a
	if err != nil {
		return err
	}
	h.Deploy(ctx, *opts)
	return nil
}

type PostAppsForm struct {
	Name string `json:"name"`
}

func (h *Server) PostApps(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	var form PostAppsForm

	if err := Decode(r, &form); err != nil {
		return err
	}

	m, err := findMessage(r)
	if err != nil {
		return err
	}

	a, err := h.Create(ctx, empire.CreateOpts{
		User:    UserFromContext(ctx),
		Name:    form.Name,
		Message: m,
	})
	if err != nil {
		return err
	}

	w.WriteHeader(201)
	return Encode(w, newApp(a))
}

func (h *Server) PatchApp(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	a, err := findApp(r, h)
	if err != nil {
		return err
	}

	var form heroku.AppUpdateOpts

	if err := Decode(r, &form); err != nil {
		return err
	}

	if form.Cert != nil {
		if err := h.CertsAttach(ctx, a, *form.Cert); err != nil {
			return err
		}
	}

	return Encode(w, newApp(a))
}

func findApp(r *http.Request, e interface {
	AppsFind(empire.AppsQuery) (*empire.App, error)
}) (*empire.App, error) {
	vars := mux.Vars(r)
	name := vars["app"]

	a, err := e.AppsFind(empire.AppsQuery{Name: &name})
	reporter.AddContext(r.Context(), "app", a.Name)
	return a, err
}
