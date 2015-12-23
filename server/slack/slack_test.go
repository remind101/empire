package slack

import (
	"io"
	"testing"

	"github.com/ejholmes/slash"
	"github.com/ejholmes/slash/slashtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

func TestCLIHandler(t *testing.T) {
	c := new(mockCLI)
	h := CLIHandler{CLI: c}

	c.On("Run", []string{
		"",
		"scale",
		"web=2",
		"-a",
		"acme-inc",
	}).Return(nil)

	r := slashtest.NewRecorder()
	err := h.ServeCommand(context.Background(), r, slash.Command{
		Text: `scale web=2 -a acme-inc`,
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
}

func TestCLIHandler_ShellWords(t *testing.T) {
	c := new(mockCLI)
	h := CLIHandler{CLI: c}

	c.On("Run", []string{
		"",
		"run",
		"sleep 60",
		"-a",
		"acme-inc",
	}).Return(nil)

	r := slashtest.NewRecorder()
	err := h.ServeCommand(context.Background(), r, slash.Command{
		Text: `run "sleep 60" -a acme-inc`,
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
}

type mockCLI struct {
	mock.Mock
}

func (m *mockCLI) Run(_ context.Context, _ io.Writer, args []string) error {
	margs := m.Called(args)
	return margs.Error(0)
}
