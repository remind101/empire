package cloudformation

import (
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/remind101/empire"
	"github.com/remind101/empire/empiretest"
	"github.com/remind101/empire/scheduler"
	"github.com/remind101/pkg/timex"
	"github.com/stretchr/testify/mock"
)

var fakeNow = time.Date(2015, time.January, 1, 1, 1, 1, 1, time.UTC)

// Stubs out time.Now in empire.
func init() {
	timex.Now = func() time.Time {
		return fakeNow
	}
}

// Run the tests with empiretest.Run, which will lock access to the database
// since it can't be shared by parallel tests.
func TestMain(m *testing.M) {
	empiretest.Run(m)
}

type mockScheduler struct {
	scheduler.Scheduler
	mock.Mock
}

func (m *mockScheduler) Submit(_ context.Context, app *scheduler.App) error {
	args := m.Called(app)
	return args.Error(0)
}

func (m *mockScheduler) Remove(_ context.Context, id string) error {
	args := m.Called(id)
	return args.Error(0)
}

type mockEmpire struct {
	empire.Empire
	mock.Mock
}

func (m *mockEmpire) Create(_ context.Context, opts empire.CreateOpts) (*empire.App, error) {
	args := m.Called(opts)
	return args.Get(0).(*empire.App), args.Error(1)
}

func (m *mockEmpire) AppsFind(q empire.AppsQuery) (*empire.App, error) {
	args := m.Called(q)
	return args.Get(0).(*empire.App), args.Error(1)
}

func (m *mockEmpire) Destroy(_ context.Context, opts empire.DestroyOpts) error {
	args := m.Called(opts)
	return args.Error(0)
}

func (m *mockEmpire) Set(_ context.Context, opts empire.SetOpts) (*empire.Config, error) {
	args := m.Called(opts)
	return args.Get(0).(*empire.Config), args.Error(1)
}
