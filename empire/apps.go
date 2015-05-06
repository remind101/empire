package empire

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/remind101/empire/empire/pkg/service"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
	"gopkg.in/gorp.v1"
)

const (
	ExposePrivate = "private"
	ExposePublic  = "public"
)

var (
	// ErrInvalidName is used to indicate that the app name is not valid.
	ErrInvalidName = &ValidationError{
		errors.New("An app name must be alphanumeric and dashes only, 3-30 chars in length."),
	}
)

// NamePattern is a regex pattern that app names must conform to.
var NamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,30}$`)

// NewAppNameFromRepo generates a new name from a Repo
//
//	remind101/r101-api => r101-api
func NewAppNameFromRepo(repo string) string {
	p := strings.Split(repo, "/")
	return p[len(p)-1]
}

// App represents an app.
type App struct {
	Name string `db:"name"`

	Repo *string `db:"repo"`

	// Valid values are empire.ExposePrivate and empire.ExposePublic.
	Exposure string `db:"exposure"`

	CreatedAt time.Time `db:"created_at"`
}

// IsValid returns an error if the app isn't valid.
func (a *App) IsValid() error {
	if !NamePattern.Match([]byte(a.Name)) {
		return ErrInvalidName
	}

	return nil
}

// PreInsert implements a pre insert hook for the db interface
func (a *App) PreInsert(s gorp.SqlExecutor) error {
	a.CreatedAt = timex.Now()

	if a.Exposure == "" {
		a.Exposure = ExposePrivate
	}

	return a.IsValid()
}

func (s *store) AppsCreate(app *App) (*App, error) {
	return appsCreate(s.db, app)
}

func (s *store) AppsUpdate(app *App) (int64, error) {
	return appsUpdate(s.db, app)
}

func (s *store) AppsDestroy(app *App) error {
	return appsDestroy(s.db, app)
}

func (s *store) AppsAll() ([]*App, error) {
	return appsAll(s.db)
}

func (s *store) AppsFind(name string) (*App, error) {
	return appsFind(s.db, name)
}

func (s *store) AppsFindByRepo(repo string) (*App, error) {
	return appsFindByRepo(s.db, repo)
}

type appsService struct {
	store   *store
	manager service.Manager
}

func (s *appsService) AppsDestroy(ctx context.Context, app *App) error {
	if err := s.manager.Remove(ctx, app.Name); err != nil {
		return err
	}

	return s.store.AppsDestroy(app)
}

// AppsEnsureRepo will set the repo if it's not set.
func (s *appsService) AppsEnsureRepo(app *App, repo string) error {
	if app.Repo != nil {
		return nil
	}

	app.Repo = &repo

	_, err := s.store.AppsUpdate(app)
	return err
}

// AppsFindOrCreateByRepo first attempts to find an app by repo, falling back to
// creating a new app.
func (s *appsService) AppsFindOrCreateByRepo(repo string) (*App, error) {
	a, err := s.store.AppsFindByRepo(repo)
	if err != nil {
		return a, err
	}

	// If the app wasn't found, create a new app linked to this repo.
	if a != nil {
		return a, nil
	}

	n := NewAppNameFromRepo(repo)

	a, err = s.store.AppsFind(n)
	if err != nil {
		return a, err
	}

	if a != nil {
		return a, s.AppsEnsureRepo(a, repo)
	}

	a = &App{
		Name: n,
		Repo: &repo,
	}

	return s.store.AppsCreate(a)
}

// AppsCreate inserts the app into the database.
func appsCreate(db *db, app *App) (*App, error) {
	return app, db.Insert(app)
}

// AppsUpdate updates an app.
func appsUpdate(db *db, app *App) (int64, error) {
	return db.Update(app)
}

// AppsDestroy destroys an app.
func appsDestroy(db *db, app *App) error {
	_, err := db.Delete(app)
	return err
}

// AppsAll returns all Apps.
func appsAll(db *db) ([]*App, error) {
	var apps []*App
	return apps, db.Select(&apps, `select * from apps order by name`)
}

// Finds an app by name.
func appsFind(db *db, name string) (*App, error) {
	return appsFindBy(db, "name", name)
}

// Finds an app by it's Repo field.
func appsFindByRepo(db *db, repo string) (*App, error) {
	return appsFindBy(db, "repo", repo)
}

// AppsFindBy finds an app by a field.
func appsFindBy(db *db, field string, value interface{}) (*App, error) {
	var app App

	if err := findBy(db, &app, "apps", field, value); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &app, nil
}

// scaler is a small service for scaling an apps process.
type scaler struct {
	store   *store
	manager service.Manager
}

func (s *scaler) Scale(ctx context.Context, app *App, t ProcessType, quantity int) error {
	release, err := s.store.ReleasesLast(app)
	if err != nil {
		return err
	}

	if release == nil {
		return &ValidationError{Err: fmt.Errorf("no releases for %s", app.Name)}
	}

	f, err := s.store.ProcessesAll(release)
	if err != nil {
		return err
	}

	p, ok := f[t]
	if !ok {
		return &ValidationError{Err: fmt.Errorf("no %s process type in release", t)}
	}

	if err := s.manager.Scale(ctx, release.AppName, string(p.Type), uint(quantity)); err != nil {
		return err
	}

	// Update quantity for this process in the formation
	p.Quantity = quantity
	_, err = s.store.ProcessesUpdate(p)
	return err
}

// restarter is a small service for restarting an apps processes.
type restarter struct {
	manager service.Manager
}

func (s *restarter) Restart(ctx context.Context, app *App, t ProcessType, id string) error {
	instances, err := s.manager.Instances(ctx, app.Name)
	if err != nil {
		return err
	}

	var selected []*service.Instance

	if id != "" {
		for _, i := range instances {
			if i.ID == id {
				selected = []*service.Instance{i}
			}
		}
	} else if t != "" {
		for _, i := range instances {
			if i.Process.Type == string(t) {
				selected = append(selected, i)
			}
		}
	}

	for _, i := range selected {
		if err := s.manager.Stop(ctx, i.ID); err != nil {
			return err
		}
	}

	return nil
}
