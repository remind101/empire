package empire

import (
	"errors"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/12factor"
	"github.com/remind101/empire/pkg/headerutil"
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

	Processes []*Process

	Description string
	CreatedAt   *time.Time
}

// Formation creates a Formation object
func (r *Release) Formation() Formation {
	f := make(Formation)
	for _, p := range r.Processes {
		f[p.Type] = p
	}
	return f
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

	// Preload all the things.
	scope = append(scope, Preload("App", "Config", "Slug", "Processes"))

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

// ReleasesFind returns the first matching release.
func (s *store) ReleasesFind(scope Scope) (*Release, error) {
	var release Release

	if err := s.First(scope, &release); err != nil {
		return &release, err
	}

	if err := s.attachPorts(&release); err != nil {
		return &release, err
	}

	return &release, nil
}

// Releases returns all releases matching the scope.
func (s *store) Releases(scope Scope) ([]*Release, error) {
	var releases []*Release
	return releases, s.Find(scope, &releases)
}

// ReleasesCreate persists a release.
func (s *store) ReleasesCreate(r *Release) (*Release, error) {
	if err := s.attachPorts(r); err != nil {
		return r, err
	}

	return releasesCreate(s.db, r)
}

// attachPorts returns a map of ports for a release. It will allocate new ports to an app if need be.
func (s *store) attachPorts(r *Release) error {
	for _, p := range r.Processes {
		if p.Type == WebProcessType {
			// TODO: Support a port per process, allowing more than one process to expose a port.
			port, err := s.PortsFindOrCreateByApp(r.App)
			if err != nil {
				return err
			}
			p.Port = port.Port
		}
	}
	return nil
}

// releasesService is a service for creating and rolling back a Release.
type releasesService struct {
	*Empire
}

// ReleasesCreate creates the release, then sets the current process formation on the release.
func (s *releasesService) ReleasesCreate(ctx context.Context, r *Release) (*Release, error) {
	// Create a new formation for this release.
	if err := s.createFormation(r); err != nil {
		return nil, err
	}

	r, err := s.store.ReleasesCreate(r)
	if err != nil {
		return r, err
	}

	// Schedule the new release onto the cluster.
	return r, s.releaser.Release(ctx, r)
}

func (s *releasesService) createFormation(release *Release) error {
	var existing Formation

	// Get the old release, so we can copy the Formation.
	last, err := s.store.ReleasesFind(ReleasesQuery{App: release.App})
	if err != nil {
		if err != gorm.RecordNotFound {
			return err
		}
	} else {
		existing = last.Formation()
	}

	f := NewFormation(existing, release.Slug.ProcessTypes)
	release.Processes = f.Processes()

	return nil
}

// Rolls back to a specific release version.
func (s *releasesService) Rollback(ctx context.Context, opts RollbackOpts) (*Release, error) {
	app, version := opts.App, opts.Version
	r, err := s.store.ReleasesFind(ReleasesQuery{App: app, Version: &version})
	if err != nil {
		return nil, err
	}

	desc := fmt.Sprintf("Rollback to v%d", version)
	return s.ReleasesCreate(ctx, &Release{
		App:         app,
		Config:      r.Config,
		Slug:        r.Slug,
		Description: desc,
	})
}

// ReleasesLastVersion returns the last ReleaseVersion for the given App. This
// function also ensures that the last release is locked until the transaction
// is commited, so the release version can be incremented atomically.
func releasesLastVersion(db *gorm.DB, appID string) (int, error) {
	var version int

	rows, err := db.Raw(`select version from releases where app_id = ? order by version desc for update`, appID).Rows()
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
	t := db.Begin()

	// Get the last release version for this app.
	v, err := releasesLastVersion(t, release.App.ID)
	if err != nil {
		t.Rollback()
		return release, err
	}

	// Increment the release version.
	release.Version = v + 1

	if err := t.Create(release).Error; err != nil {
		t.Rollback()
		return release, err
	}

	if err := t.Commit().Error; err != nil {
		t.Rollback()
		return release, err
	}

	return release, nil
}

type releaser struct {
	*Empire
}

// ScheduleRelease creates jobs for every process and instance count and
// schedules them onto the cluster.
func (r *releaser) Release(ctx context.Context, release *Release) error {
	m := newManifest(release)
	return r.Scheduler.Up(m)
}

// ReleaseApp will find the last release for an app and release it.
func (r *releaser) ReleaseApp(ctx context.Context, app *App) error {
	release, err := r.store.ReleasesFind(ReleasesQuery{App: app})
	if err != nil {
		if err == gorm.RecordNotFound {
			return ErrNoReleases
		}

		return err
	}

	if release == nil {
		return nil
	}

	return r.Release(ctx, release)
}

// newManifest turns a release into a twelvefactor.Manifest.
func newManifest(release *Release) twelvefactor.Manifest {
	app := newApp(release)

	var processes []twelvefactor.process
	for _, p := range release.Processes {
		processes = append(processes, newProcess(release, p))
	}

	return twelvefactor.Manifest{
		App:       app,
		Processes: processes,
	}
}

func newApp(release *Release) twelvefactor.App {
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

	return twelvefactor.App{
		ID:    release.App.ID,
		Name:  release.App.Name,
		Env:   environment(release.Config.Vars),
		Image: release.Slug.Image.String(),
	}
}

func newProcess(release *Release, p *Process) twelvefactor.Process {
	return twelvefactor.Process{
		Exposure: newExposure(release, p),
		Env: map[string]string{
			"EMPIRE_PROCESS": string(p.Type),
		},
		Labels: map[string]string{
			"empire.app.process": string(p.Type),
		},
		Name: string(p.Type),
		// TODO:
		// Command:      string(p.Command),
		DesiredCount: p.Quantity,
		Memory:       int(p.Constraints.Memory),
		CPUShares:    int(p.Constraints.CPUShare),
	}
}

func newExposure(release *Release, p *Process) twelvefactor.Exposure {
	switch p.Type {
	case "web":
		exposure := &twelvefactor.HTTPExposure{
			External: release.App.Exposure == ExposePublic,
		}

		if cert := release.App.Cert; cert != "" {
			return &twelvefactor.HTTPSExposure{
				HTTPExposure: *exposure,
				Cert:         cert,
			}
		}

		return exposure
	default:
		return nil
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
