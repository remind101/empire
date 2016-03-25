package empire

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
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

// AppNameFromRepo generates a name from a Repo
//
//	remind101/r101-api => r101-api
func AppNameFromRepo(repo string) string {
	p := strings.Split(repo, "/")
	return p[len(p)-1]
}

// App represents an app.
type App struct {
	ID string

	Name string

	Repo *string

	// Valid values are empire.ExposePrivate and empire.ExposePublic.
	Exposure string

	// The name of an SSL cert for the web process of this app.
	Cert string

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

// AppID returns a scope to find an app by id.
func AppID(id string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

type appsService struct {
	*Empire
}

// Destroy destroys removes an app from the scheduler, then destroys it here.
func (s *appsService) Destroy(ctx context.Context, db *gorm.DB, app *App) error {
	if err := appsDestroy(db, app); err != nil {
		return err
	}

	return s.Scheduler.Remove(ctx, app.ID)
}

func (s *appsService) Restart(ctx context.Context, db *gorm.DB, opts RestartOpts) error {
	if opts.PID != "" {
		return s.Scheduler.Stop(ctx, opts.PID)
	}

	return s.releases.ReleaseApp(ctx, db, opts.App)
}

func (s *appsService) Scale(ctx context.Context, db *gorm.DB, opts ScaleOpts) (*Process, error) {
	app, t, quantity, c := opts.App, opts.Process, opts.Quantity, opts.Constraints

	release, err := releasesFind(db, ReleasesQuery{App: app})
	if err != nil {
		return nil, err
	}

	if release == nil {
		return nil, &ValidationError{Err: fmt.Errorf("no releases for %s", app.Name)}
	}

	p := release.Process(t)
	if p == nil {
		return nil, &ValidationError{Err: fmt.Errorf("no %s process type in release", t)}
	}

	if err := s.Scheduler.Scale(ctx, release.AppID, string(p.Type), uint(quantity)); err != nil {
		return nil, err
	}

	event := opts.Event()
	event.PreviousQuantity = p.Quantity
	event.PreviousConstraints = p.Constraints

	// Update quantity for this process in the formation
	p.Quantity = quantity
	if c != nil {
		p.Constraints = *c
	}

	if err := processesUpdate(db, p); err != nil {
		return nil, err
	}

	// If there are no changes to the process size, we can do a quick scale
	// up, otherwise, we will resubmit the release to the scheduler.
	if c == nil {
		err = s.Scheduler.Scale(ctx, release.AppID, string(p.Type), uint(quantity))
	} else {
		err = s.releases.Release(ctx, release)
	}

	if err != nil {
		return p, err
	}

	return p, s.PublishEvent(event)
}

// appsEnsureRepo will set the repo if it's not set.
func appsEnsureRepo(db *gorm.DB, app *App, repo string) error {
	if app.Repo != nil {
		return nil
	}

	app.Repo = &repo

	return appsUpdate(db, app)
}

// appsFindOrCreateByRepo first attempts to find an app by repo, falling back to
// creating a new app.
func appsFindOrCreateByRepo(db *gorm.DB, repo string) (*App, error) {
	n := AppNameFromRepo(repo)
	a, err := appsFind(db, AppsQuery{Name: &n})
	if err != nil && err != gorm.RecordNotFound {
		return a, err
	}

	// If the app wasn't found, create a new app.
	if err != gorm.RecordNotFound {
		return a, appsEnsureRepo(db, a, repo)
	}

	a = &App{
		Name: n,
		Repo: &repo,
	}

	return appsCreate(db, a)
}

// appsFind finds a single app given the scope.
func appsFind(db *gorm.DB, scope Scope) (*App, error) {
	var app App
	return &app, first(db, scope, &app)
}

// apps finds all apps matching the scope.
func apps(db *gorm.DB, scope Scope) ([]*App, error) {
	var apps []*App
	// Default to ordering by name.
	scope = ComposedScope{Order("name"), scope}
	return apps, find(db, scope, &apps)
}

// appsCreate inserts the app into the database.
func appsCreate(db *gorm.DB, app *App) (*App, error) {
	return app, db.Create(app).Error
}

// appsUpdate updates an app.
func appsUpdate(db *gorm.DB, app *App) error {
	return db.Save(app).Error
}

// appsDestroy destroys an app.
func appsDestroy(db *gorm.DB, app *App) error {
	return db.Delete(app).Error
}
