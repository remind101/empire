package empire

import (
	"net/http"

	"golang.org/x/net/context"
)

// User represents a user of Empire.
type User struct {
	Name        string `json:"name"`
	GitHubToken string `json:"-"`
}

// GitHubClient returns an http.Client that will automatically add the
// GitHubToken to all requests.
func (u *User) GitHubClient() *http.Client {
	return &http.Client{
		Transport: &githubTransport{
			Token: u.GitHubToken,
		},
	}
}

// WithUser adds a user to the context.Context.
func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, UserKey, u)
}

// UserFromContext returns a user from a context.Context if one is present.
func UserFromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(UserKey).(*User)
	return u, ok
}

// githubTransport is an http.RoundTripper that will automatically set an oauth
// token as the basic auth credentials before dispatching a request.
type githubTransport struct {
	Token     string
	Transport http.RoundTripper
}

func (t *githubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Transport == nil {
		t.Transport = http.DefaultTransport
	}

	req.SetBasicAuth(t.Token, "x-oauth-basic")

	return t.Transport.RoundTrip(req)
}
