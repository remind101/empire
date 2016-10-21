package heroku

import (
	"net/http"
	"time"

	"github.com/remind101/empire/pkg/heroku"
	"github.com/remind101/empire/server/auth"
	"golang.org/x/net/context"
)

const (
	HeaderTwoFactor = "Heroku-Two-Factor-Code"
)

type Authorization heroku.OAuthAuthorization

func newAuthorization(token *AccessToken) *Authorization {
	var expIn *int
	if t := token.ExpiresAt; t != nil {
		exp := int(t.Sub(time.Now()).Seconds())
		expIn = &exp
	}
	return &Authorization{
		AccessToken: &struct {
			ExpiresIn *int   `json:"expires_in"`
			Id        string `json:"id"`
			Token     string `json:"token"`
		}{
			ExpiresIn: expIn,
			Token:     token.Token,
		},
	}
}

func (h *Server) PostAuthorizations(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	at, err := h.AccessTokensCreate(&AccessToken{
		User: auth.UserFromContext(ctx),
	})
	if err != nil {
		return err
	}

	return Encode(w, newAuthorization(at))
}
