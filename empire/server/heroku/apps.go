package heroku

import (
	"net/http"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/pkg/httpx"
	"github.com/remind101/empire/empire/pkg/reporter"
	"golang.org/x/net/context"
)

type App heroku.App

func newApp(a *empire.App) *App {
	return &App{
		Id:        a.Name,
		Name:      a.Name,
		CreatedAt: a.CreatedAt,
	}
}

func newApps(as []*empire.App) []*App {
	apps := make([]*App, len(as))

	for i := 0; i < len(as); i++ {
		apps[i] = newApp(as[i])
	}

	return apps
}

type GetApps struct {
	*empire.Empire
}

func (h *GetApps) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	apps, err := h.AppsAll()
	if err != nil {
		return err
	}

	if r, ok := reporter.FromContext(ctx); ok {
		r.Report(ctx, ErrBadRequest)
	}

	w.WriteHeader(200)
	return Encode(w, newApps(apps))
}

type DeleteApp struct {
	*empire.Empire
}

func (h *DeleteApp) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	if err := h.AppsDestroy(ctx, a); err != nil {
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

func findApp(ctx context.Context, e interface {
	AppsFind(name string) (*empire.App, error)
}) (*empire.App, error) {
	vars := httpx.Vars(ctx)
	name := vars["app"]

	a, err := e.AppsFind(name)
	if a == nil {
		return a, ErrNotFound
	}

	return a, err
}
