package cli

import (
	"bytes"
	"errors"
	"testing"

	"golang.org/x/net/context"

	"github.com/remind101/empire"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCLI_Error(t *testing.T) {
	e := new(mockEmpire)
	w := new(bytes.Buffer)
	c := newTestCLI(t, e)
	c.Writer = w

	appName := "acme-inc"
	app := &empire.App{Name: "acme-inc"}
	errBoom := errors.New("boom")

	e.On("AppsFind", empire.AppsQuery{
		Name: &appName,
	}).Return(app, errBoom)

	e.On("Restart", empire.RestartOpts{
		App: app,
	}).Return(nil)

	err := c.Run(context.Background(), []string{"emp", "restart", "-a", "acme-inc"})
	assert.Equal(t, errBoom, err)
}

func TestCLI_Tasks(t *testing.T) {
	e := new(mockEmpire)
	w := new(bytes.Buffer)
	c := newTestCLI(t, e)
	c.Writer = w

	appName := "acme-inc"
	app := &empire.App{Name: "acme-inc"}

	e.On("AppsFind", empire.AppsQuery{
		Name: &appName,
	}).Return(app, nil)

	e.On("Tasks", app).Return([]*empire.Task{
		{
			Name:        "v1.web.uuid",
			Type:        "web",
			Constraints: empire.Constraints1X,
			State:       "RUNNING",
		},
		{
			Name:        "v2.web.uuid",
			Type:        "web",
			Constraints: empire.Constraints2X,
			State:       "PENDING",
		},
	}, nil)

	err := c.Run(context.Background(), []string{"emp", "tasks", "-a", "acme-inc"})
	assert.NoError(t, err)
	assert.Equal(t, `v1.web.uuid  1X  RUNNING
v2.web.uuid  2X  PENDING
`, w.String())
}

func TestCLI_Restart(t *testing.T) {
	e := new(mockEmpire)
	w := new(bytes.Buffer)
	c := newTestCLI(t, e)
	c.Writer = w

	appName := "acme-inc"
	app := &empire.App{Name: "acme-inc"}
	user := &empire.User{}

	e.On("AppsFind", empire.AppsQuery{
		Name: &appName,
	}).Return(app, nil)

	e.On("Restart", empire.RestartOpts{
		User: user,
		App:  app,
	}).Return(nil)

	ctx := empire.WithUser(context.Background(), user)
	err := c.Run(ctx, []string{"emp", "restart", "-a", "acme-inc"})
	assert.NoError(t, err)
	assert.Equal(t, "Restarted acme-inc\n", w.String())
}

func TestCLI_RunTask(t *testing.T) {
	e := new(mockEmpire)
	w := new(bytes.Buffer)
	c := newTestCLI(t, e)
	c.Writer = w

	appName := "acme-inc"
	app := &empire.App{Name: "acme-inc"}
	user := &empire.User{}

	e.On("AppsFind", empire.AppsQuery{
		Name: &appName,
	}).Return(app, nil)

	e.On("Run", empire.RunOpts{
		User:    user,
		App:     app,
		Command: "sleep 60",
	}).Return(nil)

	ctx := empire.WithUser(context.Background(), user)
	err := c.Run(ctx, []string{"emp", "run", "sleep", "60", "-a", "acme-inc"})
	assert.NoError(t, err)
	assert.Equal(t, "Ran `sleep 60` on acme-inc, detached\n", w.String())
}

func TestCLI_Apps(t *testing.T) {
	e := new(mockEmpire)
	w := new(bytes.Buffer)
	c := newTestCLI(t, e)
	c.Writer = w

	user := &empire.User{}

	e.On("Apps", empire.AppsQuery{}).Return([]*empire.App{
		{Name: "acme-inc"},
	}, nil)

	ctx := empire.WithUser(context.Background(), user)
	err := c.Run(ctx, []string{"emp", "apps"})
	assert.NoError(t, err)
	assert.Equal(t, `acme-inc
`, w.String())
}

func TestCLI_Scale(t *testing.T) {
	e := new(mockEmpire)
	w := new(bytes.Buffer)
	c := newTestCLI(t, e)
	c.Writer = w

	appName := "acme-inc"
	user := &empire.User{}
	app := &empire.App{Name: appName}

	e.On("AppsFind", empire.AppsQuery{
		Name: &appName,
	}).Return(app, nil)

	e.On("Scale", empire.ScaleOpts{
		User:     user,
		App:      app,
		Process:  "web",
		Quantity: 2,
	}).Return(&empire.Process{}, nil)

	ctx := empire.WithUser(context.Background(), user)
	err := c.Run(ctx, []string{"emp", "scale", "web=2", "-a", "acme-inc"})
	assert.NoError(t, err)
	assert.Equal(t, `Scaled acme-inc`, w.String())
}

func newTestCLI(t testing.TB, e *mockEmpire) *CLI {
	return New(e)
}

// fatal returns an error handler that calls t.Fatal.
func fatal(t testing.TB) func(error) {
	return func(err error) {
		t.Fatal(err)
	}
}

type mockEmpire struct {
	mock.Mock
}

func (m *mockEmpire) Apps(q empire.AppsQuery) ([]*empire.App, error) {
	args := m.Called(q)
	return args.Get(0).([]*empire.App), args.Error(1)
}

func (m *mockEmpire) AppsFind(q empire.AppsQuery) (*empire.App, error) {
	args := m.Called(q)
	return args.Get(0).(*empire.App), args.Error(1)
}

func (m *mockEmpire) Restart(ctx context.Context, opts empire.RestartOpts) error {
	args := m.Called(opts)
	return args.Error(0)
}

func (m *mockEmpire) Tasks(ctx context.Context, app *empire.App) ([]*empire.Task, error) {
	args := m.Called(app)
	return args.Get(0).([]*empire.Task), args.Error(1)
}

func (m *mockEmpire) Run(ctx context.Context, opts empire.RunOpts) error {
	args := m.Called(opts)
	return args.Error(0)
}

func (m *mockEmpire) Scale(ctx context.Context, opts empire.ScaleOpts) (*empire.Process, error) {
	args := m.Called(opts)
	return args.Get(0).(*empire.Process), args.Error(1)
}
