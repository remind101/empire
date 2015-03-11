package heroku

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
)

type GetApps struct {
	Empire
}

func (h *GetApps) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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

func (h *DeleteApp) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
	Name  string `json:"name"`
	Repos struct {
		Docker *empire.Repo `json:"docker"`
		GitHub *empire.Repo `json:"github"`
	} `json:"repos"`
}

type PostApps struct {
	Empire
}

func (h *PostApps) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var form PostAppsForm

	if err := Decode(r, &form); err != nil {
		return err
	}

	app := &empire.App{
		Name: empire.AppName(form.Name),
		Repos: empire.Repos{
			Docker: form.Repos.Docker,
			GitHub: form.Repos.GitHub,
		},
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
