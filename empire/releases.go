package empire

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/empire/pkg/service"
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

// forApp returns a scope that will query only records for a certain app.
func forApp(app *App) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("app_id = ?", app.ID)
	}
}

// releasesScope returns a common scope for querying releases.
func releasesScope(app *App) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Preload("App").Preload("App.Certificate").Preload("Config").Preload("Slug").Preload("Processes").
			Scopes(forApp(app)).
			Order("version desc")
	}
}

func (s *store) ReleasesLast(app *App) (*Release, error) {
	var release Release
	if err := s.db.Scopes(releasesScope(app)).First(&release).Error; err != nil {
		if err == gorm.RecordNotFound {
			return nil, nil
		}

		return nil, err
	}
	return &release, nil
}

func (s *store) ReleasesFindByApp(app *App) ([]*Release, error) {
	var releases []*Release
	return releases, s.db.Scopes(releasesScope(app)).Find(&releases).Error
}

func (s *store) ReleasesFindByAppAndVersion(app *App, v int) (*Release, error) {
	var release Release
	if err := s.db.Scopes(releasesScope(app)).Where("version = ?", v).First(&release).Error; err != nil {
		if err == gorm.RecordNotFound {
			return nil, nil
		}

		return nil, err
	}
	return &release, nil
}

func (s *store) ReleasesCreate(r *Release) (*Release, error) {
	return releasesCreate(s.db, r)
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

	// Create port mappings for formation.
	if err := s.newProcessPorts(r); err != nil {
		return nil, err
	}

	// Schedule the new release onto the cluster.
	return r, s.releaser.Release(ctx, r)
}

func (s *releasesService) createFormation(release *Release) error {
	// Get the old release, so we can copy the Formation.
	last, err := s.store.ReleasesLast(release.App)
	if err != nil {
		return err
	}

	var existing Formation

	if last != nil {
		existing = last.Formation()
	}

	f := NewFormation(existing, release.Slug.ProcessTypes)
	release.Processes = f.Processes()

	return nil
}

// newProcessPorts returns a map of ports for a release. It will allocate new ports to an app if need be.
func (s *releasesService) newProcessPorts(r *Release) error {
	for _, p := range r.Processes {
		if p.Type == WebProcessType {
			// TODO: Support a port per process, allowing more than one process to expose a port.
			port, err := s.store.PortsFindOrCreateByApp(r.App)
			if err != nil {
				return err
			}
			p.Port = port.Port
		}
	}
	return nil
}

// Rolls back to a specific release version.
func (s *releasesService) ReleasesRollback(ctx context.Context, app *App, version int) (*Release, error) {
	r, err := s.store.ReleasesFindByAppAndVersion(app, version)
	if err != nil {
		return nil, err
	}

	if r == nil {
		return nil, &ValidationError{Err: fmt.Errorf("release %d not found", version)}
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
	manager service.Manager
}

// ScheduleRelease creates jobs for every process and instance count and
// schedules them onto the cluster.
func (r *releaser) Release(ctx context.Context, release *Release) error {
	a := newServiceApp(release)
	return r.manager.Submit(ctx, a)
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

	if len(ports) > 0 {
		env["PORT"] = fmt.Sprintf("%d", *ports[0].Container)

		// If we have exposed ports, set process exposure to apps exposure
		procExp = serviceExposure(release.App.Exposure)
	}

	cert := serviceSSLCertName(release.App.Certificate)

	return &service.Process{
		Type:        string(p.Type),
		Env:         env,
		Command:     string(p.Command),
		Image:       release.Slug.Image.String(),
		Instances:   uint(p.Quantity),
		MemoryLimit: MemoryLimit,
		CPUShares:   CPUShare,
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
		env[string(k)] = string(v)
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

func serviceSSLCertName(c *Certificate) (name string) {
	if c != nil {
		name = c.Name
	}
	return name
}
