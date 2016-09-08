package heroku

import (
	"net/http"

	"github.com/remind101/empire/server/auth"
	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

// Middleware for handling authentication.
type Authentication struct {
	auth *auth.Auth

	// handler is the wrapped httpx.Handler. This handler is called when the
	// user is authenticated.
	handler httpx.Handler
}

// Authenticat wraps an httpx.Handler in the Authentication middleware to authenticate
// the request.
func Authenticate(h httpx.Handler, auth *auth.Auth) httpx.Handler {
	return &Authentication{
		auth:    auth,
		handler: h,
	}
}

// ServeHTTPContext implements the httpx.Handler interface. It will ensure that
// there is a Bearer token present and that it is valid.
func (h *Authentication) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	username, password, ok := r.BasicAuth()
	if !ok {
		return ErrUnauthorized
	}

	ctx, err := h.auth.Authenticate(ctx, username, password, r.Header.Get(HeaderTwoFactor))
	if err != nil {
		switch err {
		case auth.ErrTwoFactor:
			return ErrTwoFactor
		case auth.ErrForbidden:
			return ErrUnauthorized
		}

		if err, ok := err.(*auth.UnauthorizedError); ok {
			return errUnauthorized(err)
		}

		return &ErrorResource{
			Status:  http.StatusForbidden,
			ID:      "forbidden",
			Message: err.Error(),
		}
	}

	// Embed the associated user into the context.
	user := auth.UserFromContext(ctx)

	logger.Info(ctx,
		"authenticated",
		"user", user.Name,
	)

	reporter.AddContext(ctx, "user", user.Name)

	return h.handler.ServeHTTPContext(ctx, w, r)
}
