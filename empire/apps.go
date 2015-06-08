package empire

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/empire/pkg/service"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
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
	ID string

	Name string

	Repo *string

	Certificates []*Certificate

	// Valid values are empire.ExposePrivate and empire.ExposePublic.
	Exposure string

	CreatedAt *time.Time
}

// IsValid returns an error if the app isn't valid.
func (a *App) IsValid() error {
	if !NamePattern.Match([]byte(a.Name)) {
		return ErrInvalidName
	}

	return nil
}

func (a *App) BeforeCreate() error {
	t := timex.Now()
	a.CreatedAt = &t

	if a.Exposure == "" {
		a.Exposure = ExposePrivate
	}

	return a.IsValid()
}

// AppsQuery is a Scope implementation for common things to filter releases
// by.
type AppsQuery struct {
	// If provided, an App ID to find.
	ID *string

	// If provided, finds apps matching the given name.
	Name *string

	// If provided, finds apps with the given repo attached.
	Repo *string
}

// Scope implements the Scope interface.
func (q AppsQuery) Scope(db *gorm.DB) *gorm.DB {
	var scope ComposedScope

	if q.ID != nil {
		scope = append(scope, ID(*q.ID))
	}

	if q.Name != nil {
		scope = append(scope, FieldEquals("name", *q.Name))
	}

	if q.Repo != nil {
		scope = append(scope, FieldEquals("repo", *q.Repo))
	}

	return scope.Scope(db)
}

// AppsFirst returns the first matching release.
func (s *store) AppsFirst(scope Scope) (*App, error) {
	var app App
	scope = ComposedScope{scope, Preload("Certificates")}
	return &app, s.First(scope, &app)
}

// Apps returns all apps matching the scope.
func (s *store) Apps(scope Scope) ([]*App, error) {
	var apps []*App
	// Default to ordering by name.
	scope = ComposedScope{Order("name"), scope}
	return apps, s.Find(scope, &apps)
}

// AppsCreate persists an app.
func (s *store) AppsCreate(app *App) (*App, error) {
	return appsCreate(s.db, app)
}

// AppsUpdate updates an app.
func (s *store) AppsUpdate(app *App) error {
	return appsUpdate(s.db, app)
}

// AppsDestroy destroys an app.
func (s *store) AppsDestroy(app *App) error {
	return appsDestroy(s.db, app)
}

// AppID returns a scope to find an app by id.
func AppID(id string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

type appsService struct {
	store   *store
	manager service.Manager
}

func (s *appsService) AppsDestroy(ctx context.Context, app *App) error {
	if err := s.manager.Remove(ctx, app.ID); err != nil {
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

	return s.store.AppsUpdate(app)
}

// AppsFindOrCreateByRepo first attempts to find an app by repo, falling back to
// creating a new app.
func (s *appsService) AppsFindOrCreateByRepo(repo string) (*App, error) {
	a, err := s.store.AppsFirst(AppsQuery{Repo: &repo})
	if err != nil && err != gorm.RecordNotFound {
		return a, err
	}

	// If the app wasn't found, create a new app linked to this repo.
	if err != gorm.RecordNotFound {
		return a, nil
	}

	n := NewAppNameFromRepo(repo)

	a, err = s.store.AppsFirst(AppsQuery{Name: &n})
	if err != nil && err != gorm.RecordNotFound {
		return a, err
	}

	if err != gorm.RecordNotFound {
		return a, s.AppsEnsureRepo(a, repo)
	}

	a = &App{
		Name: n,
		Repo: &repo,
	}

	return s.store.AppsCreate(a)
}

// AppsCreate inserts the app into the database.
func appsCreate(db *gorm.DB, app *App) (*App, error) {
	return app, db.Create(app).Error
}

// AppsUpdate updates an app.
func appsUpdate(db *gorm.DB, app *App) error {
	return db.Save(app).Error
}

// AppsDestroy destroys an app.
func appsDestroy(db *gorm.DB, app *App) error {
	return db.Delete(app).Error
}

// scaler is a small service for scaling an apps process.
type scaler struct {
	store   *store
	manager service.Manager
}

func (s *scaler) Scale(ctx context.Context, app *App, t ProcessType, quantity int) error {
	release, err := s.store.ReleasesFirst(ReleasesQuery{App: app})
	if err != nil {
		return err
	}

	if release == nil {
		return &ValidationError{Err: fmt.Errorf("no releases for %s", app.Name)}
	}

	f, err := s.store.Formation(ProcessesQuery{Release: release})
	if err != nil {
		return err
	}

	p, ok := f[t]
	if !ok {
		return &ValidationError{Err: fmt.Errorf("no %s process type in release", t)}
	}

	if err := s.manager.Scale(ctx, release.AppID, string(p.Type), uint(quantity)); err != nil {
		return err
	}

	// Update quantity for this process in the formation
	p.Quantity = quantity
	return s.store.ProcessesUpdate(p)
}

// restarter is a small service for restarting an apps processes.
type restarter struct {
	manager service.Manager
}

func (s *restarter) Restart(ctx context.Context, app *App, t ProcessType, id string) error {
	instances, err := s.manager.Instances(ctx, app.ID)
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
