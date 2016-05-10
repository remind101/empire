package empire

import (
	"errors"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/pkg/headerutil"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

var ErrNoReleases = errors.New("no releases")

// Release is a combination of a Config and a Slug, which form a deployable
// release.
type Release struct {
	ID      string
	Version int

	AppID string
	App   *App

	ConfigID string
	Config   *Config

	SlugID string
	Slug   *Slug

	Formation Formation

	Description string
	CreatedAt   *time.Time
}

// BeforeCreate sets created_at before inserting.
func (r *Release) BeforeCreate() error {
	t := timex.Now()
	r.CreatedAt = &t
	return nil
}

// ReleasesQuery is a Scope implementation for common things to filter releases
// by.
type ReleasesQuery struct {
	// If provided, an app to filter by.
	App *App

	// If provided, a version to filter by.
	Version *int

	// If provided, uses the limit and sorting parameters specified in the range.
	Range headerutil.Range
}

// Scope implements the Scope interface.
func (q ReleasesQuery) Scope(db *gorm.DB) *gorm.DB {
	var scope ComposedScope

	if app := q.App; app != nil {
		scope = append(scope, FieldEquals("app_id", app.ID))
	}

	if version := q.Version; version != nil {
		scope = append(scope, FieldEquals("version", *version))
	}

	scope = append(scope, Range(q.Range.WithDefaults(q.DefaultRange())))

	return scope.Scope(db)
}

// DefaultRange returns the default headerutil.Range used if values aren't
// provided.
func (q ReleasesQuery) DefaultRange() headerutil.Range {
	sort, order := "version", "desc"
	return headerutil.Range{
		Sort:  &sort,
		Order: &order,
	}
}

// releasesService is a service for creating and rolling back a Release.
type releasesService struct {
	*Empire
}

// Create creates a new release then submits it to the scheduler.
func (s *releasesService) Create(ctx context.Context, db *gorm.DB, r *Release) (*Release, error) {
	// Lock all releases for the given application to ensure that the
	// release version is updated automically.
	if err := db.Exec(`select 1 from releases where app_id = ? for update`, r.App.ID).Error; err != nil {
		return r, err
	}

	// During rollbacks, we can just provide the existing Formation for the
	// old release. For new releases, we need to create a new formation by
	// merging the formation from the extracted Procfile, and the Formation
	// from the existing release.
	if r.Formation == nil {
		if err := buildFormation(db, r); err != nil {
			return r, err
		}
	}

	r, err := releasesCreate(db, r)
	if err != nil {
		return r, err
	}

	// Schedule the new release onto the cluster.
	return r, s.Release(ctx, r)
}

// Rolls back to a specific release version.
func (s *releasesService) Rollback(ctx context.Context, db *gorm.DB, opts RollbackOpts) (*Release, error) {
	app, version := opts.App, opts.Version
	r, err := releasesFind(db, ReleasesQuery{App: app, Version: &version})
	if err != nil {
		return nil, err
	}

	desc := fmt.Sprintf("Rollback to v%d", version)
	return s.Create(ctx, db, &Release{
		App:         app,
		Config:      r.Config,
		Slug:        r.Slug,
		Formation:   r.Formation,
		Description: desc,
	})
}

// Release submits a release to the scheduler.
func (s *releasesService) Release(ctx context.Context, release *Release) error {
	a := newSchedulerApp(release)
	return s.Scheduler.Submit(ctx, a)
}

// ReleaseApp will find the last release for an app and release it.
func (s *releasesService) ReleaseApp(ctx context.Context, db *gorm.DB, app *App) error {
	release, err := releasesFind(db, ReleasesQuery{App: app})
	if err != nil {
		if err == gorm.RecordNotFound {
			return ErrNoReleases
		}

		return err
	}

	if release == nil {
		return nil
	}

	return s.Release(ctx, release)
}

// These associations are always available on a Release.
var releasesPreload = Preload("App", "Config", "Slug")

// releasesFind returns the first matching release.
func releasesFind(db *gorm.DB, scope Scope) (*Release, error) {
	var release Release

	scope = ComposedScope{releasesPreload, scope}
	if err := first(db, scope, &release); err != nil {
		return &release, err
	}

	return &release, nil
}

// releases returns all releases matching the scope.
func releases(db *gorm.DB, scope Scope) ([]*Release, error) {
	var releases []*Release
	scope = ComposedScope{releasesPreload, scope}
	return releases, find(db, scope, &releases)
}

func releasesUpdate(db *gorm.DB, release *Release) error {
	return db.Save(release).Error
}

func buildFormation(db *gorm.DB, release *Release) error {
	var existing Formation

	// Get the old release, so we can copy the Formation.
	last, err := releasesFind(db, ReleasesQuery{App: release.App})
	if err != nil {
		if err != gorm.RecordNotFound {
			return err
		}
	} else {
		existing = last.Formation
	}

	f, err := release.Slug.Formation()
	if err != nil {
		return err
	}
	release.Formation = f.Merge(existing)

	return nil
}

// currentFormations gets the current formations for an app
func currentFormation(db *gorm.DB, app *App) (Formation, error) {
	// Get the current release
	current, err := releasesFind(db, ReleasesQuery{App: app})
	if err != nil {
		return nil, err
	}
	f := current.Formation
	return f, nil
}

// ReleasesLastVersion returns the last ReleaseVersion for the given App.
func releasesLastVersion(db *gorm.DB, appID string) (int, error) {
	var version int

	rows, err := db.Raw(`select version from releases where app_id = ? order by version desc`, appID).Rows()
	if err != nil {
		return version, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&version)
		return version, err
	}

	return version, nil
}

// releasesCreate creates a new Release and inserts it into the database.
func releasesCreate(db *gorm.DB, release *Release) (*Release, error) {
	// Get the last release version for this app.
	v, err := releasesLastVersion(db, release.App.ID)
	if err != nil {
		return release, err
	}

	// Increment the release version.
	release.Version = v + 1

	if err := db.Create(release).Error; err != nil {
		return release, err
	}

	return release, nil
}

func newSchedulerApp(release *Release) *scheduler.App {

	var processes []*scheduler.Process

	for name, p := range release.Formation {
		processes = append(processes, newSchedulerProcess(release, name, p))
	}

	env := environment(release.Config.Vars)
	env["EMPIRE_APPID"] = release.App.ID
	env["EMPIRE_APPNAME"] = release.App.Name
	env["EMPIRE_RELEASE"] = fmt.Sprintf("v%d", release.Version)
	env["EMPIRE_CREATED_AT"] = timex.Now().Format(time.RFC3339)

	labels := map[string]string{
		"empire.app.id":      release.App.ID,
		"empire.app.name":    release.App.Name,
		"empire.app.release": fmt.Sprintf("v%d", release.Version),
	}

	return &scheduler.App{
		ID:        release.App.ID,
		Name:      release.App.Name,
		Processes: processes,
		Env:       env,
		Labels:    labels,
	}
}

func newSchedulerProcess(release *Release, name string, p Process) *scheduler.Process {
	env := map[string]string{
		"EMPIRE_PROCESS": name,
		"SOURCE":         fmt.Sprintf("%s.%s.v%d", release.App.Name, name, release.Version),
	}

	labels := map[string]string{
		"empire.app.process": name,
	}

	return &scheduler.Process{
		Type:        name,
		Env:         env,
		Labels:      labels,
		Command:     []string(p.Command),
		Image:       release.Slug.Image,
		Instances:   uint(p.Quantity),
		MemoryLimit: uint(p.Memory),
		CPUShares:   uint(p.CPUShare),
		Nproc:       uint(p.Nproc),
		Exposure:    processExposure(release.App, name),
	}
}

// environment coerces a Vars into a map[string]string.
func environment(vars Vars) map[string]string {
	env := make(map[string]string)

	for k, v := range vars {
		env[string(k)] = string(*v)
	}

	return env
}

func processExposure(app *App, process string) *scheduler.Exposure {
	// For now, only the `web` process can be exposed.
	if process != WebProcessType {
		return nil
	}

	exposure := &scheduler.Exposure{
		External: app.Exposure == ExposePublic,
	}

	switch app.Cert {
	case "":
		exposure.Type = &scheduler.HTTPExposure{}
	default:
		exposure.Type = &scheduler.HTTPSExposure{
			Cert: app.Cert,
		}
	}

	return exposure
}
