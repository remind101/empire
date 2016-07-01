package cloudformation

import (
	"database/sql"
	"errors"
	"fmt"
	"io"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/scheduler/ecs"
)

// This is the environment variable in the application that determines what step
// of the migration we should transition to. A basic migration flow would look
// like:
//
// 1. `emp set EMPIRE_SCHEDULER_MIGRATION=step1`: CloudFormation stack is
//    created without any DNS changes.
// 2. User removes the old CNAME manually in the AWS Console, then sets the
//    `DNS` parameter in the CloudFormation stack to `true`.
// 3. `emp set EMPIRE_SCHEDULER_MIGRATION=step2`: The old AWS resources are
//    removed.
// 4. `emp unset EMPIRE_SCHEDULER_MIGRATION`: All done.
const MigrationEnvVar = "EMPIRE_SCHEDULER_MIGRATION"

// ErrMigrating is returned when the application is being migrated.
var ErrMigrating = errors.New("app is currently being migrated to a CloudFormation stack. Sit tight...")

// This is a scheduler.Scheduler implementation that wraps the newer
// cloudformation.Scheduler and the older ecs.Scheduler to migrate applications
// over the the new CloudFormation based scheduler.
//
// It uses a sql table to determine what scheduling backend should be used. Apps
// can be migrated from the ecs scheduler to the cloudformation scheduler by
// using the Migrate function.
type MigrationScheduler struct {
	// The scheduler that we want to migrate to.
	cloudformation interface {
		scheduler.Scheduler
		SubmitWithOptions(context.Context, *scheduler.App, chan string, SubmitOptions) error
	}

	// The scheduler we're migrating from.
	ecs interface {
		scheduler.Scheduler
		RemoveWithOptions(context.Context, string, ecs.RemoveOptions) error
	}

	db *sql.DB
}

// NewMigrationScheduler returns a new MigrationSchedeuler instance.
func NewMigrationScheduler(db *sql.DB, c *Scheduler, e *ecs.Scheduler) *MigrationScheduler {
	return &MigrationScheduler{
		db:             db,
		cloudformation: c,
		ecs:            e,
	}
}

// Backend returns the scheduling backend to use for the given app.
func (s *MigrationScheduler) Backend(appID string) (scheduler.Scheduler, error) {
	backend, err := s.backend(appID)
	if err != nil {
		return nil, fmt.Errorf("error fetching backend: %v", err)
	}

	switch backend {
	case "ecs":
		return s.ecs, nil
	case "cloudformation":
		return s.cloudformation, nil
	default:
		return nil, ErrMigrating
	}
}

// backend returns the name of the backend to use for operations.
func (s *MigrationScheduler) backend(appID string) (string, error) {
	var backend string
	err := s.db.QueryRow(`SELECT backend FROM scheduler_migration WHERE app_id = $1`, appID).Scan(&backend)

	// For newly created apps.
	if err == sql.ErrNoRows {
		return "cloudformation", nil
	}

	return backend, err
}

func (s *MigrationScheduler) Submit(ctx context.Context, app *scheduler.App, status chan string) error {
	state, err := s.backend(app.ID)
	if err != nil {
		return err
	}

	desiredState := app.Env[MigrationEnvVar]
	if desiredState != "" {
		if err := s.Migrate(ctx, app, state, desiredState, status); err != nil {
			return fmt.Errorf("error migrating app from %s to %s: %v", state, desiredState, err)
		}
		return nil
	}

	b, err := s.Backend(app.ID)
	if err != nil {
		return err
	}
	return b.Submit(ctx, app, status)
}

// Migrate submits the app to the CloudFormation scheduler, waits for the stack
// to successfully create, then removes the old API managed resources using the
// ECS scheduler.
func (s *MigrationScheduler) Migrate(ctx context.Context, app *scheduler.App, state, desiredState string, status chan string) error {
	errTransition := fmt.Errorf("cannot transition from %s to %s", state, desiredState)

	// Whether or not we're re-trying a state transition.
	rerun := state == desiredState

	switch desiredState {
	case "step1":
		if !rerun && state != "ecs" {
			return errTransition
		}

		// Submit to cloudformation and wait for it to complete successfully.
		// Don't make any DNS changes.
		if err := s.cloudformation.SubmitWithOptions(ctx, app, status, SubmitOptions{
			NoDNS: aws.Bool(true),
		}); err != nil {
			return fmt.Errorf("error creating CloudFormation stack: %v", err)
		}

		// After this step, the user has a couple of options.
		//
		// 1. The user can proceed by migrating to step2
		// 2. The user can remove the old CNAME, then update the DNS
		//    parameter in the CloudFormation stack to `true`.

		state = "step1"
	case "step2":
		if !rerun && state != "step1" {
			return errTransition
		}

		// The user may have already manually enabled the DNS change,
		// but let's make sure.
		if err := s.cloudformation.SubmitWithOptions(ctx, app, status, SubmitOptions{
			NoDNS: aws.Bool(false),
		}); err != nil {
			return fmt.Errorf("error updating CloudFormation stack: %v", err)
		}

		// Remove the old AWS resources.
		if err := s.ecs.RemoveWithOptions(ctx, app.ID, ecs.RemoveOptions{
			NoDNS: true,
		}); err != nil {
			return fmt.Errorf("error removing existing ECS resources: %v", err)
		}

		state = "cloudformation"
	default:
		return fmt.Errorf("cannot transition to %s", desiredState)
	}

	_, err := s.db.Exec(`UPDATE scheduler_migration SET backend = $1 WHERE app_id = $2`, state, app.ID)
	return err
}

func (s *MigrationScheduler) Remove(ctx context.Context, appID string) error {
	b, err := s.Backend(appID)
	if err != nil {
		return err
	}
	if err := b.Remove(ctx, appID); err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM scheduler_migration WHERE app_id = $1`, appID)
	return err
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

func (s *MigrationScheduler) Stop(ctx context.Context, id string) error {
	// These are identical between the old and new scheduler, so just using
	// the new one is safe.
	return s.cloudformation.Stop(ctx, id)
}
