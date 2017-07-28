package heroku

import (
	"net/http"

	"github.com/remind101/empire/pkg/heroku"
	"github.com/remind101/empire/server/auth"
)

const (
	HeaderTwoFactor = "Heroku-Two-Factor-Code"
)

type Authorization heroku.OAuthAuthorization

func newAuthorization(token *AccessToken) *Authorization {
	var expIn *int
	if t := token.ExpiresAt; t != nil {
		exp := int(token.ExpiresIn().Seconds())
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

func (h *Server) PostAuthorizations(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	session := auth.SessionFromContext(ctx)

	at, err := h.AccessTokensCreate(&AccessToken{
		User:      session.User,
		ExpiresAt: session.ExpiresAt,
	})
	if err != nil {
		return err
	}

	return Encode(w, newAuthorization(at))
}
