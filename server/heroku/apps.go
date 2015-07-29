package heroku

import (
	"net/http"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire"
	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

type App heroku.App

func newApp(a *empire.App) *App {
	return &App{
		Id:        a.ID,
		Name:      a.Name,
		CreatedAt: *a.CreatedAt,
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
	apps, err := h.Apps(empire.AppsQuery{})
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newApps(apps))
}

type GetAppInfo struct {
	*empire.Empire
}

func (h *GetAppInfo) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	w.WriteHeader(200)
	return Encode(w, newApp(a))
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

type DeployApp struct {
	*empire.Empire
}

func (h *DeployApp) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	a, err := findApp(ctx, h)
	if err != nil {
		return err
	}

	opts, err := getDeploymentsCreateOpts(ctx, w, r)
	opts.App = a
	if err != nil {
		return err
	}
	h.Deploy(ctx, opts)
	return nil
}

type PostAppsForm struct {
	Name string  `json:"name"`
	Repo *string `json:"repo"`
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
		Repo: form.Repo,
	}
	a, err := h.AppsCreate(app)
	if err != nil {
		return err
	}

	w.WriteHeader(201)
	return Encode(w, newApp(a))
}

func findApp(ctx context.Context, e interface {
	AppsFirst(empire.AppsQuery) (*empire.App, error)
}) (*empire.App, error) {
	vars := httpx.Vars(ctx)
	name := vars["app"]

	a, err := e.AppsFirst(empire.AppsQuery{Name: &name})
	reporter.AddContext(ctx, "app", a.Name)
	return a, err
}
