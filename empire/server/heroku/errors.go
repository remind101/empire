package heroku

import (
	"net/http"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/empire"
)

// Named matching heroku's error codes. See
// https://devcenter.heroku.com/articles/platform-api-reference#error-responses
var (
	ErrBadRequest = &ErrorResource{
		Status:  http.StatusBadRequest,
		ID:      "bad_request",
		Message: "Request invalid, validate usage and try again",
	}
	ErrUnauthorized = &ErrorResource{
		Status:  http.StatusUnauthorized,
		ID:      "unauthorized",
		Message: "Request not authenticated, API token is missing, invalid or expired",
	}
	ErrForbidden = &ErrorResource{
		Status:  http.StatusForbidden,
		ID:      "forbidden",
		Message: "Request not authorized, provided credentials do not provide access to specified resource",
	}
	ErrNotFound = &ErrorResource{
		Status:  http.StatusNotFound,
		ID:      "not_found",
		Message: "Request failed, the specified resource does not exist",
	}
	ErrTwoFactor = &ErrorResource{
		Status:  http.StatusUnauthorized,
		ID:      "two_factor",
		Message: "Two factor code is required.",
	}
)

// ErrorResource represents the error response format that we return.
type ErrorResource struct {
	Status  int    `json:"-"`
	ID      string `json:"id"`
	Message string `json:"message"`
	URL     string `json:"url"`
}

func newError(err error) *ErrorResource {
	if err == gorm.RecordNotFound {
		return ErrNotFound
	}

	switch err := err.(type) {
	case *ErrorResource:
		return err
	case *empire.ValidationError:
		return ErrBadRequest
	default:
		return &ErrorResource{
			Message: err.Error(),
		}
	}
}

// Error implements error interface.
func (e *ErrorResource) Error() string {
	return e.Message
}

func errNotImplemented(message string) *ErrorResource {
	return &ErrorResource{
		Status:  http.StatusNotImplemented,
		ID:      "not_implemented",
		Message: message,
	}
}
