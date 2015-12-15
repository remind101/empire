package empire

import "net/http"

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
