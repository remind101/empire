package heroku

import (
	"net/http"

	"github.com/bgentry/heroku-go"
	"github.com/gorilla/mux"
	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
)

type App heroku.App

func newApp(app *empire.App) *App {
	return &App{
		Id:        app.Name,
		Name:      app.Name,
		CreatedAt: app.CreatedAt,
	}
}

func newApps(apps []*empire.App) []*App {
	happs := make([]*App, len(apps))

	for i := 0; i < len(apps); i++ {
		happs[i] = newApp(apps[i])
	}

	return happs
}

type GetApps struct {
	*empire.Empire
}

func (h *GetApps) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	apps, err := h.AppsAll()
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newApps(apps))
}

type DeleteApp struct {
	*empire.Empire
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
	*empire.Empire
}

func (h *PostApps) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var form PostAppsForm

	if err := Decode(r, &form); err != nil {
		return err
	}

	app := &empire.App{
		Name: form.Name,
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
	return Encode(w, newApp(a))
}

func findApp(r *http.Request, e interface {
	AppsFind(name string) (*empire.App, error)
}) (*empire.App, error) {
	vars := mux.Vars(r)
	name := vars["app"]

	a, err := e.AppsFind(name)
	if a == nil {
		return a, ErrNotFound
	}

	return a, err
}
