package heroku

import (
	"net/http"

	"github.com/remind101/empire/server/auth"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/reporter"
)

// Authenticat authenticates the request. If the user is not authenticated, an
// error is returned. If the request is authenticated, the User is embedded in
// the requets context.Context.
func (h *Server) Authenticate(r *http.Request) (*http.Request, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return r, ErrUnauthorized
	}

	user, err := h.Authenticator.Authenticate(username, password, r.Header.Get(HeaderTwoFactor))
	if err != nil {
		switch err {
		case auth.ErrTwoFactor:
			return r, ErrTwoFactor
		case auth.ErrForbidden:
			return r, ErrUnauthorized
		}

		if err, ok := err.(*auth.UnauthorizedError); ok {
			return r, errUnauthorized(err)
		}

		return r, &ErrorResource{
			Status:  http.StatusForbidden,
			ID:      "forbidden",
			Message: err.Error(),
		}
	}

	// Embed the associated user into the context.
	r = r.WithContext(WithUser(r.Context(), user))

	logger.Info(r.Context(),
		"authenticated",
		"user", user.Name,
	)

	reporter.AddContext(r.Context(), "user", user.Name)

	return r, nil
}
