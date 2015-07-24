package empire

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/pkg/headerutil"
	"github.com/remind101/empire/pkg/service"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

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

func (r *Release) Formation() Formation {
	f := make(Formation)
	for _, p := range r.Processes {
		f[p.Type] = p
	}
	return f
}

// Set created_at before inserting.
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
	Range *headerutil.Range
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

	if r := q.Range; r != nil {
		scope = append(scope, Range(r))
	} else {
		scope = append(scope, Order("version desc"))
	}

	// Preload all the things.
	scope = append(scope, Preload("App", "Config", "Slug", "Processes"))

	return scope.Scope(db)
}

// ReleasesFirst returns the first matching release.
func (s *store) ReleasesFirst(scope Scope) (*Release, error) {
	var release Release
	// TODO: Wrap the store with this. Gorm blows up when preloading
	// App.Certificates on a collection of releases.
	scope = ComposedScope{scope, Preload("App.Certificates")}
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
	store    *store
	releaser *releaser
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
	last, err := s.store.ReleasesFirst(ReleasesQuery{App: release.App})
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
func (s *releasesService) ReleasesRollback(ctx context.Context, app *App, version int) (*Release, error) {
	r, err := s.store.ReleasesFirst(ReleasesQuery{App: app, Version: &version})
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
	store   *store
	manager service.Manager
}

// ScheduleRelease creates jobs for every process and instance count and
// schedules them onto the cluster.
func (r *releaser) Release(ctx context.Context, release *Release) error {
	a := newServiceApp(release)
	return r.manager.Submit(ctx, a)
}

// ReleaseApp will find the last release for an app and release it.
func (r *releaser) ReleaseApp(ctx context.Context, app *App) error {
	release, err := r.store.ReleasesFirst(ReleasesQuery{App: app})
	if err != nil {
		return err
	}

	if release == nil {
		return nil
	}

	return r.Release(ctx, release)
}

func newServiceApp(release *Release) *service.App {
	var processes []*service.Process

	for _, p := range release.Processes {
		processes = append(processes, newServiceProcess(release, p))
	}

	return &service.App{
		ID:        release.App.ID,
		Name:      release.App.Name,
		Processes: processes,
	}
}

func newServiceProcess(release *Release, p *Process) *service.Process {
	var procExp service.Exposure
	ports := newServicePorts(int64(p.Port))

	env := environment(release.Config.Vars)
	env["EMPIRE_APPNAME"] = release.App.Name
	env["EMPIRE_PROCESS"] = string(p.Type)
	env["EMPIRE_RELEASE"] = fmt.Sprintf("v%d", release.Version)
	env["EMPIRE_CREATED_AT"] = timex.Now().Format(time.RFC3339)
	env["SOURCE"] = fmt.Sprintf("%s.v%d.%s", release.App.Name, release.Version, p.Type)

	if len(ports) > 0 {
		env["PORT"] = fmt.Sprintf("%d", *ports[0].Container)

		// If we have exposed ports, set process exposure to apps exposure
		procExp = serviceExposure(release.App.Exposure)
	}

	cert := serviceSSLCertName(release.App.Certificates)

	return &service.Process{
		Type:        string(p.Type),
		Env:         env,
		Command:     string(p.Command),
		Image:       release.Slug.Image,
		Instances:   uint(p.Quantity),
		MemoryLimit: uint(p.Constraints.Memory),
		CPUShares:   uint(p.Constraints.CPUShare),
		Ports:       ports,
		Exposure:    procExp,
		SSLCert:     cert,
	}
}

func newServicePorts(hostPort int64) []service.PortMap {
	var ports []service.PortMap
	if hostPort != 0 {
		// TODO: We can just map the same host port as the container port, as we make it
		// available as $PORT in the env vars.
		port := int64(WebPort)
		ports = append(ports, service.PortMap{
			Host:      &hostPort,
			Container: &port,
		})
	}
	return ports
}

// environment coerces a Vars into a map[string]string.
func environment(vars Vars) map[string]string {
	env := make(map[string]string)

	for k, v := range vars {
		env[string(k)] = string(*v)
	}

	return env
}

func serviceExposure(appExp string) (exp service.Exposure) {
	switch appExp {
	case ExposePrivate:
		exp = service.ExposePrivate
	case ExposePublic:
		exp = service.ExposePublic
	default:
		exp = service.ExposeNone
	}

	return exp
}

func serviceSSLCertName(certs []*Certificate) (name string) {
	if len(certs) > 0 {
		name = certs[0].Name
	}
	return name
}
