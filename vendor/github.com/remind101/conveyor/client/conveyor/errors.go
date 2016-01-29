package conveyor

import "net/url"

var (
	ErrNotFound = &Error{
		ID:      "not_found",
		Message: "resource was not found",
	}
)

func notFound(err error) bool {
	if err, ok := err.(*url.Error); ok {
		if err, ok := err.Err.(*Error); ok {
			return err.ID == ErrNotFound.ID
		}
	}

	return false
}
