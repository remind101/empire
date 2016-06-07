package cloudformation

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/remind101/empire/scheduler"
	"github.com/remind101/empire/scheduler/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// When DefaultNew is false, we should default to the old ECS scheduler for new
// apps.
func TestMigrationScheduler_NewApp_ECS(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	e := new(mockECSScheduler)
	c := new(mockCloudFormationScheduler)
	s := &MigrationScheduler{
		Default:        "ecs",
		ecs:            e,
		cloudformation: c,
		db:             db,
	}

	app := &scheduler.App{
		ID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Processes: []*scheduler.Process{
			{Type: "web"},
		},
	}

	e.On("Submit", app).Return(nil)

	err := s.Submit(context.Background(), app)
	assert.NoError(t, err)

	e.AssertExpectations(t)
	c.AssertExpectations(t)
}

// When DefaultNew is true, we should default to the new CloudFormation
// scheduler.
func TestMigrationScheduler_NewApp_CloudFormation(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	e := new(mockECSScheduler)
	c := new(mockCloudFormationScheduler)
	s := &MigrationScheduler{
		Default:        "cloudformation",
		ecs:            e,
		cloudformation: c,
		db:             db,
	}

	app := &scheduler.App{
		ID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Processes: []*scheduler.Process{
			{Type: "web"},
		},
	}

	c.On("Submit", app).Return(nil)

	err := s.Submit(context.Background(), app)
	assert.NoError(t, err)

	e.AssertExpectations(t)
	c.AssertExpectations(t)
}

// Old apps that aren't being migrated should just be routed to the ECS
// scheduler.
func TestMigrationScheduler_OldApp(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	e := new(mockECSScheduler)
	c := new(mockCloudFormationScheduler)
	s := &MigrationScheduler{
		ecs:            e,
		cloudformation: c,
		db:             db,
	}

	_, err := db.Exec(`INSERT INTO scheduler_migration (app_id, backend) VALUES ('c9366591-ab68-4d49-a333-95ce5a23df68', 'ecs')`)
	assert.NoError(t, err)

	app := &scheduler.App{
		ID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Processes: []*scheduler.Process{
			{Type: "web"},
		},
	}

	e.On("Submit", app).Return(nil)

	err = s.Submit(context.Background(), app)
	assert.NoError(t, err)

	e.AssertExpectations(t)
	c.AssertExpectations(t)
}

func TestMigrationScheduler_Migrate(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	e := new(mockECSScheduler)
	c := new(mockCloudFormationScheduler)
	s := &MigrationScheduler{
		ecs:            e,
		cloudformation: c,
		db:             db,
	}

	_, err := db.Exec(`INSERT INTO scheduler_migration (app_id, backend) VALUES ('c9366591-ab68-4d49-a333-95ce5a23df68', 'ecs')`)
	assert.NoError(t, err)

	app := &scheduler.App{
		ID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Env: map[string]string{
			MigrationEnvVar: "step1",
		},
		Processes: []*scheduler.Process{
			{
				Type: "web",
			},
		},
	}

	// Step1: Create the CloudFormation stack without making any DNS
	// changes.
	c.On("SubmitWithOptions", app, SubmitOptions{
		NoDNS: true,
	}).Return(nil)

	err = s.Submit(context.Background(), app)
	assert.NoError(t, err)

	e.AssertExpectations(t)
	c.AssertExpectations(t)

	// Step2: Update the CloudFormation stack with the DNS changes, and
	// remove the existing ECS resources.
	app.Env[MigrationEnvVar] = "step2"

	c.On("Submit", app).Return(nil)
	e.On("RemoveWithOptions", app.ID, ecs.RemoveOptions{
		NoDNS: true,
	}).Return(nil)

	err = s.Submit(context.Background(), app)
	assert.NoError(t, err)

	e.AssertExpectations(t)
	c.AssertExpectations(t)

	// Step3: Finalize the migration.
	delete(app.Env, MigrationEnvVar)

	c.On("Submit", app).Return(err)

	err = s.Submit(context.Background(), app)
	assert.NoError(t, err)

	e.AssertExpectations(t)
	c.AssertExpectations(t)
}

// It's not unlikely that the first couple of migrations will get rolled back,
// because of Empire configuration issues (not having the correct permissions).
//
// To account for that, it should be possible to run step1 multiple times.
func TestMigrationScheduler_Migrate_Rollback(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	e := new(mockECSScheduler)
	c := new(mockCloudFormationScheduler)
	s := &MigrationScheduler{
		ecs:            e,
		cloudformation: c,
		db:             db,
	}

	_, err := db.Exec(`INSERT INTO scheduler_migration (app_id, backend) VALUES ('c9366591-ab68-4d49-a333-95ce5a23df68', 'ecs')`)
	assert.NoError(t, err)

	app := &scheduler.App{
		ID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Env: map[string]string{
			MigrationEnvVar: "step1",
		},
		Processes: []*scheduler.Process{
			{
				Type: "web",
			},
		},
	}

	c.On("SubmitWithOptions", app, SubmitOptions{
		NoDNS: true,
	}).Return(nil).Twice()

	err = s.Submit(context.Background(), app)
	assert.NoError(t, err)

	// Let's assume the the CloudFormation stack that was created got rolled
	// back, so they manually delete the stack and try again.
	err = s.Submit(context.Background(), app)
	assert.NoError(t, err)

	e.AssertExpectations(t)
	c.AssertExpectations(t)
}

func TestMigrationScheduler_Migrate_InvalidStateTransitions(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	e := new(mockECSScheduler)
	c := new(mockCloudFormationScheduler)
	s := &MigrationScheduler{
		ecs:            e,
		cloudformation: c,
		db:             db,
	}

	_, err := db.Exec(`INSERT INTO scheduler_migration (app_id, backend) VALUES ('c9366591-ab68-4d49-a333-95ce5a23df68', 'ecs')`)
	assert.NoError(t, err)

	app := &scheduler.App{
		ID: "c9366591-ab68-4d49-a333-95ce5a23df68",
		Env: map[string]string{
			MigrationEnvVar: "step2",
		},
		Processes: []*scheduler.Process{
			{
				Type: "web",
			},
		},
	}

	err = s.Submit(context.Background(), app)
	assert.Error(t, err)
	assert.EqualError(t, err, "error migrating app from ecs to step2: cannot transition from ecs to step2")

	app.Env[MigrationEnvVar] = "step3"

	err = s.Submit(context.Background(), app)
	assert.Error(t, err)
	assert.EqualError(t, err, "error migrating app from ecs to step3: cannot transition to step3")

	e.AssertExpectations(t)
	c.AssertExpectations(t)
}

type mockScheduler struct {
	scheduler.Scheduler
	mock.Mock
}

func (m *mockScheduler) Submit(_ context.Context, app *scheduler.App) error {
	args := m.Called(app)
	return args.Error(0)
}

type mockECSScheduler struct {
	mockScheduler
}

func (m *mockECSScheduler) RemoveWithOptions(_ context.Context, appID string, opts ecs.RemoveOptions) error {
	args := m.Called(appID, opts)
	return args.Error(0)
}

type mockCloudFormationScheduler struct {
	mockScheduler
}

func (m *mockCloudFormationScheduler) SubmitWithOptions(_ context.Context, app *scheduler.App, opts SubmitOptions) error {
	args := m.Called(app, opts)
	return args.Error(0)
}
