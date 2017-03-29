package empire

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/pkg/headerutil"
	"github.com/remind101/empire/procfile"
	"github.com/remind101/empire/twelvefactor"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

// Release is a combination of a Config and a Slug, which form a deployable
// release. Releases are generally considered immutable, the only operation that
// changes a release is when altering the Quantity or Constraints inside the
// Formation.
type Release struct {
	// A unique uuid to identify this release.
	ID string

	// An auto incremented ID for this release, scoped to the application.
	Version int

	// The id of the application that this release relates to.
	AppID string

	// The application that this release relates to.
	App *App

	// The id of the config that this release uses.
	ConfigID string

	// The config that this release uses.
	Config *Config

	// The id of the slug that this release uses.
	SlugID string

	// The Slug that this release uses.
	Slug *Slug

	// The process formation to use.
	Formation Formation

	// A description for the release. Usually contains the reason for why
	// the release was created (e.g. deployment, config changes, etc).
	Description string

	// The time that this release was created.
	CreatedAt *time.Time
}

// Procfile returns the Procfile that generated this Release.
func (r *Release) Procfile() (procfile.Procfile, error) {
	return r.Slug.ParsedProcfile()
}

// BeforeCreate sets created_at before inserting.
func (r *Release) BeforeCreate() error {
	t := timex.Now()
	r.CreatedAt = &t
	return nil
}

// ReleasesQuery is a scope implementation for common things to filter releases
// by.
type ReleasesQuery struct {
	// If provided, an app to filter by.
	App *App

	// If provided, a version to filter by.
	Version *int

	// If provided, uses the limit and sorting parameters specified in the range.
	Range headerutil.Range
}

// scope implements the scope interface.
func (q ReleasesQuery) scope(db *gorm.DB) *gorm.DB {
	var scope composedScope

	if app := q.App; app != nil {
		scope = append(scope, fieldEquals("app_id", app.ID))
	}

	if version := q.Version; version != nil {
		scope = append(scope, fieldEquals("version", *version))
	}

	scope = append(scope, inRange(q.Range.WithDefaults(q.DefaultRange())))

	return scope.scope(db)
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

// CreateAndRelease creates a new release then submits it to the scheduler.
func (s *releasesService) CreateAndRelease(ctx context.Context, db *gorm.DB, r *Release, ss twelvefactor.StatusStream) (*Release, error) {
	r, err := s.Create(ctx, db, r)
	if err != nil {
		return r, err
	}
	// Schedule the new release onto the cluster.
	return r, s.Release(ctx, r, ss)
}

// Create creates a new release.
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

	return releasesCreate(db, r)
}

// Rolls back to a specific release version.
func (s *releasesService) Rollback(ctx context.Context, db *gorm.DB, opts RollbackOpts) (*Release, error) {
	app, version := opts.App, opts.Version
	r, err := releasesFind(db, ReleasesQuery{App: app, Version: &version})
	if err != nil {
		return nil, err
	}

	desc := fmt.Sprintf("Rollback to v%d", version)
	desc = appendMessageToDescription(desc, opts.User, opts.Message)
	return s.CreateAndRelease(ctx, db, &Release{
		App:         app,
		Config:      r.Config,
		Slug:        r.Slug,
		Formation:   r.Formation,
		Description: desc,
	}, nil)
}

// Release submits a release to the scheduler.
func (s *releasesService) Release(ctx context.Context, release *Release, ss twelvefactor.StatusStream) error {
	a, err := newSchedulerApp(release)
	if err != nil {
		return err
	}
	return s.Scheduler.Submit(ctx, a, ss)
}

// Restart will find the last release for an app and submit it to the scheduler
// to restart the app.
func (s *releasesService) Restart(ctx context.Context, db *gorm.DB, app *App) error {
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

	a, err := newSchedulerApp(release)
	if err != nil {
		return err
	}
	return s.Scheduler.Restart(ctx, a, nil)
}

// These associations are always available on a Release.
var releasesPreload = preload("App", "Config", "Slug")

// releasesFind returns the first matching release.
func releasesFind(db *gorm.DB, scope scope) (*Release, error) {
	var release Release

	scope = composedScope{releasesPreload, scope}
	if err := first(db, scope, &release); err != nil {
		return &release, err
	}

	return &release, nil
}

// releases returns all releases matching the scope.
func releases(db *gorm.DB, scope scope) ([]*Release, error) {
	var releases []*Release
	scope = composedScope{releasesPreload, scope}
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

func newSchedulerApp(release *Release) (*twelvefactor.Manifest, error) {
	var processes []*twelvefactor.Process

	for name, p := range release.Formation {
		if p.NoService {
			// If the entry is marked as "NoService", don't send it
			// to the backend.
			continue
		}

		if p.Quantity < 0 {
			// If the process is scaled to a negative value, don't
			// send it to the backend.
			continue
		}

		process, err := newSchedulerProcess(release, name, p)
		if err != nil {
			return nil, err
		}
		processes = append(processes, process)
	}

	env := environment(release.Config.Vars)
	env["EMPIRE_APPID"] = release.App.ID
	env["EMPIRE_APPNAME"] = release.App.Name
	env["EMPIRE_RELEASE"] = fmt.Sprintf("v%d", release.Version)

	labels := map[string]string{
		"empire.app.id":      release.App.ID,
		"empire.app.name":    release.App.Name,
		"empire.app.release": fmt.Sprintf("v%d", release.Version),
	}

	return &twelvefactor.Manifest{
		AppID:     release.App.ID,
		Name:      release.App.Name,
		Release:   fmt.Sprintf("v%d", release.Version),
		Processes: processes,
		Env:       env,
		Labels:    labels,
	}, nil
}

func newSchedulerProcess(release *Release, name string, p Process) (*twelvefactor.Process, error) {
	env := make(map[string]string)
	for k, v := range p.Environment {
		env[k] = v
	}

	env["EMPIRE_PROCESS"] = name
	env["EMPIRE_PROCESS_SCALE"] = fmt.Sprintf("%d", p.Quantity)
	env["SOURCE"] = fmt.Sprintf("%s.%s.v%d", release.App.Name, name, release.Version)

	labels := map[string]string{
		"empire.app.process": name,
	}

	var (
		exposure *twelvefactor.Exposure
		err      error
	)
	// For `web` processes defined in the standard procfile, we'll
	// generate a default exposure setting and also set the PORT
	// environment variable for backwards compatability.
	if name == webProcessType && len(p.Ports) == 0 {
		exposure = standardWebExposure(release.App)
		env["PORT"] = "8080"
	} else {
		exposure, err = processExposure(release.App, name, p)
		if err != nil {
			return nil, err
		}
	}

	return &twelvefactor.Process{
		Type:      name,
		Env:       env,
		Labels:    labels,
		Command:   []string(p.Command),
		Image:     release.Slug.Image,
		Quantity:  p.Quantity,
		Memory:    uint(p.Memory),
		CPUShares: uint(p.CPUShare),
		Nproc:     uint(p.Nproc),
		Exposure:  exposure,
		Schedule:  processSchedule(name, p),
	}, nil
}

// environment coerces a Vars into a map[string]string.
func environment(vars Vars) map[string]string {
	env := make(map[string]string)

	for k, v := range vars {
		env[string(k)] = string(*v)
	}

	return env
}

// standardWebExposure generates a scheduler.Exposure for a web process in the
// standard Procfile format.
func standardWebExposure(app *App) *twelvefactor.Exposure {
	ports := []twelvefactor.Port{
		{
			Container: 8080,
			Host:      80,
			Protocol:  &twelvefactor.HTTP{},
		},
	}

	// If a certificate is attached to the "web" process, add an SSL port.
	if cert, ok := app.Certs[webProcessType]; ok {
		ports = append(ports, twelvefactor.Port{
			Container: 8080,
			Host:      443,
			Protocol: &twelvefactor.HTTPS{
				Cert: cert,
			},
		})
	}

	return &twelvefactor.Exposure{
		External: app.Exposure == exposePublic,
		Ports:    ports,
	}
}

func processExposure(app *App, name string, process Process) (*twelvefactor.Exposure, error) {
	// No ports == not exposed
	if len(process.Ports) == 0 {
		return nil, nil
	}

	var ports []twelvefactor.Port
	for _, p := range process.Ports {
		var protocol twelvefactor.Protocol
		switch p.Protocol {
		case "http":
			protocol = &twelvefactor.HTTP{}
		case "https":
			cert, ok := app.Certs[name]
			if !ok {
				return nil, &NoCertError{Process: name}
			}
			protocol = &twelvefactor.HTTPS{
				Cert: cert,
			}
		case "tcp":
			protocol = &twelvefactor.TCP{}
		case "ssl":
			cert, ok := app.Certs[name]
			if !ok {
				return nil, &NoCertError{Process: name}
			}
			protocol = &twelvefactor.SSL{
				Cert: cert,
			}
		}
		ports = append(ports, twelvefactor.Port{
			Host:      p.Host,
			Container: p.Container,
			Protocol:  protocol,
		})
	}
	return &twelvefactor.Exposure{
		External: app.Exposure == exposePublic,
		Ports:    ports,
	}, nil
}

func processSchedule(name string, p Process) twelvefactor.Schedule {
	if p.Cron != nil {
		return twelvefactor.CRONSchedule(*p.Cron)
	}

	return nil
}

func appendMessageToDescription(main string, user *User, message string) string {
	var formatted string
	if message != "" {
		formatted = fmt.Sprintf(": '%s'", message)
	}
	return fmt.Sprintf("%s (%s%s)", main, user.Name, formatted)
}
