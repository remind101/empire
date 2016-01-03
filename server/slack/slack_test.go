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
	h := CLIHandler{
		NewCLI: func(w io.Writer) CLI {
			c.w = w
			return c
		},
	}

	c.On("Run", []string{
		"",
		"scale",
		"web=2",
		"-a",
		"acme-inc",
	}).Return("Scaling", nil)

	r := slashtest.NewRecorder()
	err := h.ServeCommand(context.Background(), r, slash.Command{
		Command: "/emp",
		Text:    `scale web=2 -a acme-inc`,
	})
	assert.NoError(t, err)

	select {
	case resp := <-r.Responses:
		assert.Equal(t, "/emp scale web=2 -a acme-inc:\n```Scaling```", resp.Text)
	default:
		t.Fatal("no responses")
	}

	c.AssertExpectations(t)
}

func TestCLIHandler_ShellWords(t *testing.T) {
	c := new(mockCLI)
	h := CLIHandler{NewCLI: func(io.Writer) CLI { return c }}

	c.On("Run", []string{
		"",
		"run",
		"sleep 60",
		"-a",
		"acme-inc",
	}).Return("", nil)

	r := slashtest.NewRecorder()
	err := h.ServeCommand(context.Background(), r, slash.Command{
		Text: `run "sleep 60" -a acme-inc`,
	})
	assert.NoError(t, err)

	select {
	case <-r.Responses:
		t.Fatal("Expected no response")
	default:
	}

	c.AssertExpectations(t)
}

type mockCLI struct {
	mock.Mock
	w io.Writer
}

func (m *mockCLI) Run(_ context.Context, args []string) error {
	margs := m.Called(args)
	if m.w != nil {
		io.WriteString(m.w, margs.String(0))
	}
	return margs.Error(1)
}
