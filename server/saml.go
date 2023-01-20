package server

import (
	"net/http"
	"io"
	"net/url"
	"text/template"

	"github.com/remind101/empire"
	"github.com/remind101/empire/internal/saml"
	samlauth "github.com/remind101/empire/server/auth/saml"
	"github.com/remind101/empire/server/heroku"
	"github.com/remind101/pkg/reporter"
)

// SAMLLogin starts a Service Provider initiated login. It generates an
// AuthnRequest, signs the generated id and stores it in a cookie, then starts
// the login with the IdP.
func (s *Server) SAMLLogin(w http.ResponseWriter, r *http.Request) {
	if s.ServiceProvider == nil {
		http.NotFound(w, r)
		return
	}

	// TODO(ejholmes): Handle error
	_ = s.ServiceProvider.InitiateLogin(w)
}

// SAMLACS handles the SAML Response call. It will validate the SAML Response
// and assertions, generate an API token, then present the token to the user.
func (s *Server) SAMLACS(w http.ResponseWriter, r *http.Request) {
	if s.ServiceProvider == nil {
		http.NotFound(w, r)
		return
	}

	assertion, err := s.ServiceProvider.Parse(w, r)
	if err != nil {
		if err, ok := err.(*saml.InvalidResponseError); ok {
			reporter.Report(r.Context(), err.PrivateErr)
		}
		http.Error(w, err.Error(), 403)
		return
	}

	session := samlauth.SessionFromAssertion(assertion)

	// Create an Access Token for the API.
	at, err := s.Heroku.AccessTokensCreate(&heroku.AccessToken{
		ExpiresAt: session.ExpiresAt,
		User:      session.User,
	})
	if err != nil {
		http.Error(w, err.Error(), 403)
		return
	}

	switch r.Header.Get("Accept") {
	case "text/plain":
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, at.Token)
	default:
		w.Header().Set("Content-Type", "text/html")
		instructionsTemplate.Execute(w, &instructionsData{
			URL:   s.URL,
			User:  session.User,
			Token: at,
		})
	}
}

type instructionsData struct {
	URL   *url.URL
	User  *empire.User
	Token *heroku.AccessToken
}

var instructionsTemplate = template.Must(template.New("instructions").Parse(`
<html>
<head>
<style>
pre.terminal {
  background-color: #444;
  color: #eee;
  padding: 20px;
  margin: 100px;
  overflow-x: scroll;
  border-radius: 4px;
}
</style>
</head>
<body>
<pre class="terminal">
<code>$ export EMPIRE_API_URL="{{.URL}}"
$ emp logout
$ cat &lt;&lt;EOF &gt;&gt; ~/.netrc # Expires in {{.Token.ExpiresIn}}
machine {{.URL.Host}}
  login {{.User.Name}}
  password {{.Token.Token}}
EOF</code>
</pre>
</body>
</html>
`))
