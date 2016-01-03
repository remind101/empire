package slash

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const testForm = `token=abcd&team_id=T012A0ABC&team_domain=acme&channel_id=D012A012A&channel_name=directmessage&user_id=U012A012A&user_name=ejholmes&command=%2Fdeploy&text=acme-inc+to+staging&response_url=https://hooks.slack.com/commands/1234/5678`

func TestCommandFromValues(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", strings.NewReader(testForm))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	assert.NoError(t, req.ParseForm())

	u, err := url.Parse("https://hooks.slack.com/commands/1234/5678")
	assert.NoError(t, err)

	cmd, err := CommandFromValues(req.Form)
	assert.NoError(t, err)
	assert.Equal(t, Command{
		Token:       "abcd",
		TeamID:      "T012A0ABC",
		TeamDomain:  "acme",
		ChannelID:   "D012A012A",
		ChannelName: "directmessage",
		UserID:      "U012A012A",
		UserName:    "ejholmes",
		Command:     "/deploy",
		Text:        "acme-inc to staging",
		ResponseURL: u,
	}, cmd)
}

func TestParseRequest(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", strings.NewReader(testForm))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	u, err := url.Parse("https://hooks.slack.com/commands/1234/5678")
	assert.NoError(t, err)

	got, err := ParseRequest(req)
	assert.NoError(t, err)
	assert.Equal(t, got, Command{
		Token:       "abcd",
		TeamID:      "T012A0ABC",
		TeamDomain:  "acme",
		ChannelID:   "D012A012A",
		ChannelName: "directmessage",
		UserID:      "U012A012A",
		UserName:    "ejholmes",
		Command:     "/deploy",
		Text:        "acme-inc to staging",
		ResponseURL: u,
	})
}

type mockHandler struct {
	mock.Mock
}

func (h *mockHandler) ServeCommand(ctx context.Context, r Responder, command Command) error {
	args := h.Called(ctx, r, command)
	return args.Error(1)
}

type mockResponder struct {
	mock.Mock
}

func (r *mockResponder) Respond(resp Response) error {
	args := r.Called(resp)
	return args.Error(0)
}
