package empire

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/remind101/empire/empire/pkg/service"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
	"gopkg.in/gorp.v1"
)

// Release is a combination of a Config and a Slug, which form a deployable
// release.
type Release struct {
	ID  string `db:"id"`
	Ver int    `db:"version"` // Version conflicts with gorps optimistic locking.

	AppID    string `db:"app_id"`
	ConfigID string `db:"config_id"`
	SlugID   string `db:"slug_id"`

	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}

// PreInsert implements a pre insert hook for the db interface
func (r *Release) PreInsert(s gorp.SqlExecutor) error {
	r.CreatedAt = timex.Now()
	return nil
}

// ReleasesCreateOpts represents options that can be passed when creating a
// new Release.
type ReleasesCreateOpts struct {
	App         *App
	Config      *Config
	Slug        *Slug
	Description string
}

func (s *store) ReleasesLast(app *App) (*Release, error) {
	return releasesLast(s.db, app.ID)
}

func (s *store) ReleasesFindByApp(app *App) ([]*Release, error) {
	return releasesAllByAppID(s.db, app.ID)
}

func (s *store) ReleasesFindByAppAndVersion(app *App, v int) (*Release, error) {
	return releasesFindByAppIDAndVersion(s.db, app.ID, v)
}

func (s *store) ReleasesFindByAppIDAndVersion(appID string, v int) (*Release, error) {
	return releasesFindByAppIDAndVersion(s.db, appID, v)
}

func (s *store) ReleasesCreate(opts ReleasesCreateOpts) (*Release, error) {
	release := &Release{
		AppID:       opts.App.ID,
		ConfigID:    opts.Config.ID,
		SlugID:      opts.Slug.ID,
		Description: opts.Description,
	}
	return releasesCreate(s.db, release)
}

// releasesService is a service for creating and rolling back a Release.
type releasesService struct {
	store    *store
	releaser *releaser
}

// ReleasesCreate creates the release, then sets the current process formation on the release.
func (s *releasesService) ReleasesCreate(ctx context.Context, opts ReleasesCreateOpts) (*Release, error) {
	app, config, slug := opts.App, opts.Config, opts.Slug

	r, err := s.store.ReleasesCreate(opts)
	if err != nil {
		return r, err
	}

	// Create a new formation for this release.
	formation, err := s.createFormation(r, slug)
	if err != nil {
		return nil, err
	}

	// Create port mappings for formation.
	ports, err := s.newProcessPorts(r, formation)
	if err != nil {
		return nil, err
	}

	// Schedule the new release onto the cluster.
	if err := s.releaser.Release(ctx, app, r, config, slug, formation, ports, serviceExposure(opts.App.Exposure)); err != nil {
		return r, err
	}

	return r, nil
}

func (s *releasesService) createFormation(release *Release, slug *Slug) (Formation, error) {
	// Get the old release, so we can copy the Formation.
	prev := release.Ver - 1
	last, err := s.store.ReleasesFindByAppIDAndVersion(release.AppID, prev)
	if err != nil {
		return nil, err
	}

	var existing Formation

	if last != nil {
		existing, err = s.store.ProcessesAll(last)
		if err != nil {
			return nil, err
		}
	}

	f := NewFormation(existing, slug.ProcessTypes)

	for _, p := range f {
		p.ReleaseID = release.ID

		if _, err := s.store.ProcessesCreate(p); err != nil {
			return f, err
		}
	}

	return f, nil
}

// newProcessPorts returns a map of ports for a release. It will allocate new ports to an app if need be.
func (s *releasesService) newProcessPorts(release *Release, formation Formation) (ProcessPortMap, error) {
	m := ProcessPortMap{}
	for _, p := range formation {
		if p.Type == WebProcessType {
			// TODO: Support a port per process, allowing more than one process to expose a port.
			port, err := s.store.PortsFindOrCreateByApp(&App{ID: release.AppID})
			if err != nil {
				return m, err
			}
			m[WebProcessType] = int64(port.Port)
		}
	}
	return m, nil
}

// Rolls back to a specific release version.
func (s *releasesService) ReleasesRollback(ctx context.Context, app *App, version int) (*Release, error) {
	prevRelease, err := s.store.ReleasesFindByAppAndVersion(app, version)
	if err != nil {
		return nil, err
	}

	if prevRelease == nil {
		return nil, &ValidationError{Err: fmt.Errorf("release %d not found", version)}
	}

	config, err := s.store.ConfigsFind(prevRelease.ConfigID)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, &ValidationError{Err: fmt.Errorf("config %s not found", prevRelease.ConfigID)}
	}

	// Find slug
	slug, err := s.store.SlugsFind(prevRelease.SlugID)
	if err != nil {
		return nil, err
	}

	if slug == nil {
		return nil, &ValidationError{Err: fmt.Errorf("slug %s not found", prevRelease.SlugID)}
	}

	desc := fmt.Sprintf("Rollback to v%d", version)
	release, err := s.ReleasesCreate(ctx, ReleasesCreateOpts{
		App:         app,
		Config:      config,
		Slug:        slug,
		Description: desc,
	})
	if err != nil {
		return release, err
	}

	return release, nil
}

// ReleasesFindByAppIDAndVersion finds a specific version of a release for a
// given app.
func releasesFindByAppIDAndVersion(db *db, appID string, v int) (*Release, error) {
	var release Release

	if err := db.SelectOne(&release, `select * from releases where app_id = $1 and version = $2 limit 1`, appID, v); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &release, nil
}

// releasesCreate creates a new Release and inserts it into the database.
func releasesCreate(db *db, release *Release) (*Release, error) {
	t, err := db.Begin()
	if err != nil {
		return release, err
	}

	// Get the last release version for this app.
	v, err := releasesLastVersion(t, release.AppID)
	if err != nil {
		return release, err
	}

	// Increment the release version.
	release.Ver = v + 1

	if err := t.Insert(release); err != nil {
		return release, err
	}

	return release, t.Commit()
}

// ReleasesLastVersion returns the last ReleaseVersion for the given App. This
// function also ensures that the last release is locked until the transaction
// is commited, so the release version can be incremented atomically.
func releasesLastVersion(db interface {
	SelectOne(interface{}, string, ...interface{}) error
}, appID string) (version int, err error) {
	err = db.SelectOne(&version, `select version from releases where app_id = $1 order by version desc for update`, string(appID))

	if err == sql.ErrNoRows {
		return 0, nil
	}

	return
}

// ReleasesLast returns the last Release for the given App.
func releasesLast(db *db, appID string) (*Release, error) {
	var release Release

	if err := db.SelectOne(&release, `select * from releases where app_id = $1 order by version desc limit 1`, appID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &release, nil
}

// ReleasesFindByAppID finds the latest release for the given app.
func releasesAllByAppID(db *db, appID string) ([]*Release, error) {
	var rs []*Release
	return rs, db.Select(&rs, `select * from releases where app_id = $1 order by version desc`, appID)
}

type releaser struct {
	manager service.Manager
}

// ScheduleRelease creates jobs for every process and instance count and
// schedules them onto the cluster.
func (r *releaser) Release(ctx context.Context, app *App, release *Release, config *Config, slug *Slug, formation Formation, ports ProcessPortMap, exposure service.Exposure) error {
	a := newServiceApp(app, release, config, slug, formation, ports, exposure)
	return r.manager.Submit(ctx, a)
}

func newServiceApp(app *App, release *Release, config *Config, slug *Slug, formation Formation, ports ProcessPortMap, exposure service.Exposure) *service.App {
	var processes []*service.Process

	for _, p := range formation {
		processes = append(processes, newServiceProcess(release, config, slug, p, ports[p.Type], exposure))
	}

	return &service.App{
		ID:        release.AppID,
		Name:      app.Name,
		Processes: processes,
	}
}

func newServiceProcess(release *Release, config *Config, slug *Slug, p *Process, port int64, exposure service.Exposure) *service.Process {
	var procExp service.Exposure
	ports := newServicePorts(port)
	env := environment(config.Vars)

	if len(ports) > 0 {
		env["PORT"] = fmt.Sprintf("%d", *ports[0].Container)

		// If we have exposed ports, set process exposure to apps exposure
		procExp = exposure
	}

	return &service.Process{
		Type:        string(p.Type),
		Env:         env,
		Command:     string(p.Command),
		Image:       slug.Image.String(),
		Instances:   uint(p.Quantity),
		MemoryLimit: MemoryLimit,
		CPUShares:   CPUShare,
		Ports:       ports,
		Exposure:    procExp,
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
