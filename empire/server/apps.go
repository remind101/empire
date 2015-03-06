package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/remind101/empire/empire"
)

type GetApps struct {
	Empire
}

func (h *GetApps) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	apps, err := h.AppsAll()
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, apps)
}

type DeleteApp struct {
	Empire
}

func (h *DeleteApp) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(r, h)
	if err != nil {
		return err
	}

	if err := h.AppsDestroy(a); err != nil {
		return err
	}

	return NoContent(w)
}

type PostAppsForm struct {
	Name string `json:"name"`
	Repo string `json:"repo"`
}

type PostApps struct {
	Empire
}

func (h *PostApps) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	var form PostAppsForm

	if err := Decode(r, &form); err != nil {
		return err
	}

	dockerRepo := empire.Repo(form.Repo)
	app := &empire.App{
		Name: empire.AppName(form.Name),
		Repos: empire.Repos{
			Docker: &dockerRepo,
		},
	}
	if err := app.IsValid(); err != nil {
		return ErrBadRequest
	}

	a, err := h.AppsCreate(app)
	if err != nil {
		return err
	}

	w.WriteHeader(201)
	return Encode(w, a)
}

func findApp(r *http.Request, e empire.AppsFinder) (*empire.App, error) {
	vars := mux.Vars(r)
	name := vars["app"]

	a, err := e.AppsFind(empire.AppName(name))
	if a == nil {
		return a, ErrNotFound
	}

	return a, err
}
