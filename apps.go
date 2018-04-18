package empire

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/pkg/timex"
	"golang.org/x/net/context"
)

const (
	exposePrivate = "private"
	exposePublic  = "public"
)

// NamePattern is a regex pattern that app names must conform to.
var NamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,30}$`)

// appNameFromRepo generates a name from a Repo
//
//	remind101/r101-api => r101-api
func appNameFromRepo(repo string) string {
	p := strings.Split(repo, "/")
	return p[len(p)-1]
}

// Certs maps a process name to a certificate to use for any SSL listeners.
type Certs map[string]string

// Scan implements the sql.Scanner interface.
func (c *Certs) Scan(src interface{}) error {
	bytes, ok := src.([]byte)
	if !ok {
		return error(errors.New("Scan source was not []bytes"))
	}

	certs := make(Certs)
	if err := json.Unmarshal(bytes, &certs); err != nil {
		return err
	}
	*c = certs

	return nil
}

// Value implements the driver.Value interface.
func (c Certs) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}

	raw, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	return driver.Value(raw), nil
}

// App represents an Empire application.
type App struct {
	// A unique uuid that identifies the application.
	ID string

	// The name of the application.
	Name string

	// If provided, the Docker repo that this application is linked to.
	// Deployments to Empire, which don't specify an application, will use
	// this field to determine what app an image should be deployed to.
	Repo *string

	// Valid values are exposePrivate and exposePublic.
	Exposure string

	// Maps a process name to an SSL certificate to use for the SSL listener
	// of the load balancer.
	Certs Certs

	// The time that this application was created.
	CreatedAt *time.Time

	// Maintenance defines whether the app is in maintenance mode or not.
	Maintenance bool
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
		a.Exposure = exposePrivate
	}

	return a.IsValid()
}

// AppsQuery is a scope implementation for common things to filter releases
// by.
type AppsQuery struct {
	// If provided, an App ID to find.
	ID *string

	// If provided, finds apps matching the given name.
	Name *string

	// If provided, finds apps with the given repo attached.
	Repo *string
}

// scope implements the scope interface.
func (q AppsQuery) scope(db *gorm.DB) *gorm.DB {
	var scope composedScope

	if q.ID != nil {
		scope = append(scope, idEquals(*q.ID))
	}

	if q.Name != nil {
		scope = append(scope, fieldEquals("name", *q.Name))
	}

	if q.Repo != nil {
		scope = append(scope, fieldEquals("repo", *q.Repo))
	}

	return scope.scope(db)
}

type appsService struct {
	*Empire
}

// Destroy destroys removes an app from the scheduler, then destroys it here.
func (s *appsService) Destroy(ctx context.Context, db Storage, app *App) error {
	if err := db.AppsDestroy(app); err != nil {
		return err
	}

	return s.Scheduler.Remove(ctx, app.ID)
}

func (s *appsService) Restart(ctx context.Context, db *gorm.DB, opts RestartOpts) error {
	if opts.PID != "" {
		return s.Scheduler.Stop(ctx, opts.PID)
	}

	return s.releases.Restart(ctx, db, opts.App)
}

func (s *appsService) Scale(ctx context.Context, db *gorm.DB, opts ScaleOpts) ([]*Process, error) {
	app := opts.App

	release, err := releasesFind(db, ReleasesQuery{App: app})
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, &ValidationError{Err: fmt.Errorf("no releases for %s", app.Name)}
	}

	event := opts.Event()

	var ps []*Process
	for i, up := range opts.Updates {
		t, q, c := up.Process, up.Quantity, up.Constraints

		p, ok := release.Formation[t]
		if !ok {
			return nil, &ValidationError{Err: fmt.Errorf("no %s process type in release", t)}
		}

		eventUpdate := event.Updates[i]
		eventUpdate.PreviousQuantity = p.Quantity
		eventUpdate.PreviousConstraints = p.Constraints()

		// Update quantity for this process in the formation
		p.Quantity = q
		if c != nil {
			p.SetConstraints(*c)
		}

		release.Formation[t] = p
		ps = append(ps, &p)
	}

	// Save the new formation.
	if err := releasesUpdate(db, release); err != nil {
		return nil, err
	}

	err = s.releases.Release(ctx, release, nil)
	if err != nil {
		return ps, err
	}

	return ps, s.PublishEvent(event)
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
	n := appNameFromRepo(repo)
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
func appsFind(db *gorm.DB, scope scope) (*App, error) {
	var app App
	return &app, first(db, scope, &app)
}

// apps finds all apps matching the scope.
func apps(db *gorm.DB, scope scope) ([]*App, error) {
	var apps []*App
	// Default to ordering by name.
	scope = composedScope{order("name"), scope}
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
