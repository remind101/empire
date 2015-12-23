package slash

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
)

func TestMux_Command_Found(t *testing.T) {
	r := new(mockResponder)
	h := new(mockHandler)
	m := NewMux()
	m.Command("/deploy", "token", h)

	cmd := Command{
		Token:   "token",
		Command: "/deploy",
	}

	ctx := context.Background()
	h.On("ServeCommand",
		WithParams(ctx, make(map[string]string)),
		r,
		cmd,
	).Return(Reply(""), nil)

	err := m.ServeCommand(ctx, r, cmd)
	assert.NoError(t, err)

	h.AssertExpectations(t)
}

func TestMux_Command_NotFound(t *testing.T) {
	r := new(mockResponder)
	m := NewMux()

	cmd := Command{
		Command: "/deploy",
	}

	ctx := context.Background()
	err := m.ServeCommand(ctx, r, cmd)
	assert.Equal(t, err, ErrNoHandler)
}

func TestMux_MatchText_Found(t *testing.T) {
	r := new(mockResponder)
	h := new(mockHandler)
	m := NewMux()
	m.MatchText(regexp.MustCompile(`(?P<repo>\S+?) to (?P<environment>\S+?)$`), h)

	cmd := Command{
		Text: "acme-inc to staging",
	}

	ctx := context.Background()
	h.On("ServeCommand",
		WithParams(ctx, map[string]string{"repo": "acme-inc", "environment": "staging"}),
		r,
		cmd,
	).Return(Reply(""), nil)

	err := m.ServeCommand(ctx, r, cmd)
	assert.NoError(t, err)

	h.AssertExpectations(t)
}

func TestValidateToken(t *testing.T) {
	r := new(mockResponder)
	h := new(mockHandler)
	a := ValidateToken(h, "foo")

	ctx := context.Background()
	err := a.ServeCommand(ctx, r, Command{})
	assert.Equal(t, ErrInvalidToken, err)

	cmd := Command{
		Token: "foo",
	}
	h.On("ServeCommand", ctx, r, cmd).Return(Reply(""), nil)
	err = a.ServeCommand(ctx, r, cmd)
	assert.NoError(t, err)
	h.AssertExpectations(t)
}

func TestValidateToken_Empty(t *testing.T) {
	r := new(mockResponder)
	h := new(mockHandler)
	a := ValidateToken(h, "")

	ctx := context.Background()
	err := a.ServeCommand(ctx, r, Command{})
	assert.Equal(t, ErrInvalidToken, err)
}

func TestMatchTextRegexp(t *testing.T) {
	re := regexp.MustCompile(`(?P<repo>\S+?) to (?P<environment>\S+?)(!)?$`)
	m := MatchTextRegexp(re)

	_, ok := m.Match(Command{Text: "foo"})
	assert.False(t, ok)

	params, ok := m.Match(Command{Text: "acme-inc to staging"})
	assert.True(t, ok)
	assert.Equal(t, map[string]string{"repo": "acme-inc", "environment": "staging"}, params)

	params, ok = m.Match(Command{Text: "acme-inc to staging!"})
	assert.True(t, ok)
	assert.Equal(t, map[string]string{"repo": "acme-inc", "environment": "staging"}, params)
}

func TestMatchSubcommand(t *testing.T) {
	m := MatchSubcommand("help")

	_, ok := m.Match(Command{Text: "foo"})
	assert.False(t, ok)

	_, ok = m.Match(Command{Text: "foo help"})
	assert.False(t, ok)

	_, ok = m.Match(Command{Text: "help"})
	assert.True(t, ok)

	_, ok = m.Match(Command{Text: "help with something"})
	assert.True(t, ok)
}

func TestResponder(t *testing.T) {
	var called bool
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		raw, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, `{"text":"ok"}`, string(raw))
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	r := &responder{
		responseURL: u,
		client:      http.DefaultClient,
	}

	err := r.Respond(Reply("ok"))
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestResponder_Err(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		io.WriteString(w, "Used url")
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	r := &responder{
		responseURL: u,
		client:      http.DefaultClient,
	}

	err := r.Respond(Reply("ok"))
	assert.EqualError(t, err, "error sending delayed response: Used url")
}
