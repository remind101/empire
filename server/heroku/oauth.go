package heroku

import (
	"net/http"

	"github.com/remind101/empire/pkg/heroku"
	"github.com/remind101/empire"
	"golang.org/x/net/context"
)

const (
	HeaderTwoFactor = "Heroku-Two-Factor-Code"
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
}

func (h *PostAuthorizations) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	at, err := h.Empire.AccessTokensCreate(&empire.AccessToken{
		User: UserFromContext(ctx),
	})
	if err != nil {
		return err
	}

	return Encode(w, newAuthorization(at))
}
