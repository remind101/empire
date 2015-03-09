package server

import (
	"net/http"

	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
)

// Middleware for handling authentication.
type Authentication struct {
	finder  empire.AccessTokensFinder
	handler Handler
}

// Authenticat wraps a Handler in the Authentication middleware to authenticate
// the request.
func Authenticate(f empire.AccessTokensFinder, h Handler) Handler {
	return &Authentication{
		finder:  f,
		handler: h,
	}
}

// ServeHTTPContext implements the Handler interface. It will ensure that
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
