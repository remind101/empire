package heroku

import (
	"net/http"

	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/pkg/httpx"
	"golang.org/x/net/context"
)

type TokenFinder interface {
	AccessTokensFind(string) (*empire.AccessToken, error)
}

// Middleware for handling authentication.
type Authentication struct {
	finder  TokenFinder
	handler httpx.Handler
}

// Authenticat wraps an httpx.Handler in the Authentication middleware to authenticate
// the request.
func Authenticate(f TokenFinder, h httpx.Handler) httpx.Handler {
	return &Authentication{
		finder:  f,
		handler: h,
	}
}

// ServeHTTPContext implements the httpx.Handler interface. It will ensure that
// there is a Bearer token present and that it is valid.
func (h *Authentication) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	token, ok := extractToken(r)
	if !ok {
		return ErrUnauthorized
	}

	at, err := h.finder.AccessTokensFind(token)
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
