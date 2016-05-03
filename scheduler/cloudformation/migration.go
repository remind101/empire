package cloudformation

import (
	"database/sql"
	"fmt"
	"io"
	"time"

	"golang.org/x/net/context"

	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/scheduler/ecs"
)

// This is a scheduler.Scheduler implementation that wraps the newer
// cloudformation.Scheduler and the older ecs.Scheduler to migrate applications
// over the the new CloudFormation based scheduler.
//
// It uses a sql table to determine what scheduling backend should be used. Apps
// can be migrated from the ecs scheduler to the cloudformation scheduler by
// using the Migrate function.
type MigrationScheduler struct {
	// This is the amount of time to wait for the new CloudFormation stack
	// to stabilize before swapping over to it. 5-10 minutes is probably
	// enough for the load balancer to starting being resolvable.
	WaitTime time.Duration

	// The scheduler that we want to migrate to.
	cloudformation *Scheduler

	// The scheduler we're migrating from.
	ecs *ecs.Scheduler

	db *sql.DB
}

// NewMigrationScheduler returns a new MigrationSchedeuler instance.
func NewMigrationScheduler(db *sql.DB, c *Scheduler, e *ecs.Scheduler) *MigrationScheduler {
	return &MigrationScheduler{
		WaitTime:       10 * time.Minute,
		db:             db,
		cloudformation: c,
		ecs:            e,
	}
}

// Backend returns the scheduling backend to use for the given app.
func (s *MigrationScheduler) Backend(appID string) (scheduler.Scheduler, error) {
	backend, err := s.backend(appID)
	if err != nil {
		return nil, err
	}

	switch backend {
	case "ecs":
		return s.ecs, nil
	case "cloudformation":
		return s.cloudformation, nil
	case "migrate":
		return nil, fmt.Errorf("%s is currently being migrated to a CloudFormation stack. Sit tight...", appID)
	default:
		return nil, fmt.Errorf("unexpected scheduling backend encountered: %s", backend)
	}
}

// backend returns the name of the backend to use for operations.
func (s *MigrationScheduler) backend(appID string) (string, error) {
	var name string
	err := s.db.QueryRow(`SELECT backend FROM scheduler_migration WHERE app_id = $1`, appID).Scan(&name)

	// For newly created apps.
	if err == sql.ErrNoRows {
		return "cloudformation", nil
	}

	return name, err
}

// Migrate prepares this app to be migrated to the cloudformation backend. The
// next time the app is deployed, it will be deployed to using the
// cloudformation backend, then the old resources will be removed.
func (s *MigrationScheduler) Prepare(appID string) error {
	_, err := s.db.Exec(`UPDATE scheduler_migration SET backend = 'migrate' WHERE app_id = $1`, appID)
	return err
}

func (s *MigrationScheduler) Submit(ctx context.Context, app *scheduler.App) error {
	backend, err := s.backend(app.ID)
	if err != nil {
		return err
	}

	if backend == "migrate" {
		return s.Migrate(ctx, app)
	}

	b, err := s.Backend(app.ID)
	if err != nil {
		return err
	}
	return b.Submit(ctx, app)
}

// Migrate submits the app to the CloudFormation scheduler, waits for the stack
// to successfully create, then removes the old API managed resources using the
// ECS scheduler.
func (s *MigrationScheduler) Migrate(ctx context.Context, app *scheduler.App) error {
	// Submit to cloudformation and wait for it to complete successfully.
	// Don't make any DNS changes.
	if err := s.cloudformation.SubmitWithOptions(ctx, app, SubmitOptions{
		Wait:  true,
		NoDNS: true,
	}); err != nil {
		return err
	}

	// Wait for some time for everything to start running.
	<-time.After(s.WaitTime)

	// Remove the old AWS resources.
	if err := s.ecs.Remove(ctx, app.ID); err != nil {
		return err
	}

	// Finalize the changes by altering DNS.
	if err := s.cloudformation.SubmitWithOptions(ctx, app, SubmitOptions{
		Wait: true,
	}); err != nil {
		return err
	}

	_, err := s.db.Exec(`UPDATE scheduler_migration SET backend = 'cloudformation' WHERE app_id = $1`, app.ID)
	return err
}

func (s *MigrationScheduler) Remove(ctx context.Context, appID string) error {
	b, err := s.Backend(appID)
	if err != nil {
		return err
	}
	return b.Remove(ctx, appID)
}

func (s *MigrationScheduler) Instances(ctx context.Context, appID string) ([]*scheduler.Instance, error) {
	b, err := s.Backend(appID)
	if err != nil {
		return nil, err
	}
	return b.Instances(ctx, appID)
}

func (s *MigrationScheduler) Run(ctx context.Context, app *scheduler.App, process *scheduler.Process, in io.Reader, out io.Writer) error {
	b, err := s.Backend(app.ID)
	if err != nil {
		return err
	}
	return b.Run(ctx, app, process, in, out)
}

func (s *MigrationScheduler) Scale(ctx context.Context, appID, process string, instances uint) error {
	b, err := s.Backend(appID)
	if err != nil {
		return err
	}
	return b.Scale(ctx, appID, process, instances)
}

func (s *MigrationScheduler) Stop(ctx context.Context, id string) error {
	// These are identical between the old and new scheduler, so just using
	// the new one is safe.
	return s.cloudformation.Stop(ctx, id)
}
