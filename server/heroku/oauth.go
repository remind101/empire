package heroku

import (
	"fmt"
	"net/http"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire"
	"github.com/remind101/empire/server/authorization"
	"golang.org/x/net/context"
)

const (
	HeaderTwoFactor       = "Heroku-Two-Factor-Code"
	HeaderGitHubTwoFactor = "X-GitHub-OTP"
)

type Authorization heroku.OAuthAuthorization

func newAuthorization(token *empire.AccessToken) *Authorization {
	return &Authorization{
		AccessToken: &struct {
			ExpiresIn *int   `json:"expires_in"`
			Id        string `json:"id"`
			Token     string `json:"token"`
		}{
			Token: token.Token,
		},
	}
}

type PostAuthorizations struct {
	*empire.Empire
	authorization.Authorizer
}

func (h *PostAuthorizations) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return ErrBadRequest
	}

	u, err := h.Authorize(user, pass, r.Header.Get(HeaderTwoFactor))
	if err != nil {
		switch err {
		case authorization.ErrTwoFactor:
			return ErrTwoFactor
		case authorization.ErrUnauthorized:
			return ErrUnauthorized
		}

		msg := err.Error()
		if err, ok := err.(*authorization.MembershipError); ok {
			msg = fmt.Sprintf("You are not a member of %s", err.Organization)
		}

		return &ErrorResource{
			Status:  http.StatusForbidden,
			ID:      "forbidden",
			Message: msg,
		}
	}

	at, err := h.Empire.AccessTokensCreate(&empire.AccessToken{
		User: u,
	})
	if err != nil {
		return err
	}

	return Encode(w, newAuthorization(at))
}
