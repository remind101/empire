package heroku

import (
	"net/http"

	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/pkg/httpx"
	"golang.org/x/net/context"
)

// Middleware for handling authentication.
type Authentication struct {
	// findAccessToken is a function that, given a string token, will return
	// an empire.AccessToken
	findAccessToken func(string) (*empire.AccessToken, error)

	// handler is the wrapped httpx.Handler. This handler is called when the
	// user is authenticated.
	handler httpx.Handler
}

// Authenticat wraps an httpx.Handler in the Authentication middleware to authenticate
// the request.
func Authenticate(e *empire.Empire, h httpx.Handler) httpx.Handler {
	return &Authentication{
		findAccessToken: e.AccessTokensFind,
		handler:         h,
	}
}

// ServeHTTPContext implements the httpx.Handler interface. It will ensure that
// there is a Bearer token present and that it is valid.
func (h *Authentication) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	token, ok := extractToken(r)
	if !ok {
		return ErrUnauthorized
	}

	at, err := h.findAccessToken(token)
	if err != nil {
		return err
	}

	// Token is invalid or not found.
	if at == nil {
		return ErrUnauthorized
	}

	// Embed the associated user into the context.
	ctx = empire.WithUser(ctx, at.User)

	return h.handler.ServeHTTPContext(ctx, w, r)
}

// extractToken extracts an AccessToken Token from a request.
func extractToken(r *http.Request) (string, bool) {
	_, pass, ok := r.BasicAuth()
	if !ok {
		return "", false
	}

	return pass, true
}
