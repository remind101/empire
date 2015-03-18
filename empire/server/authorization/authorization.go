package authorization

import (
	"errors"
	"fmt"

	"github.com/remind101/empire/empire"
)

var (
	// ErrTwoFactor is returned by an Authorizer when a two factor code is
	// either invalid or required.
	ErrTwoFactor = errors.New("authorization: two factor code required or invalid")

	// ErrUnauthorized is returned when creating an authorization fails
	// because of invalid credentials.
	ErrUnauthorized = errors.New("authorization: unauthorized")
)

// Authorizer is an interface that can authorize a user.
type Authorizer interface {
	Authorize(username, password, twofactor string) (*empire.User, error)
}

// Fake is a fake implementation of the Authorizer interface that let's
// anyone in. Used in development.
type Fake struct{}

// Authorizer implements Authorizer Authorize.
func (a *Fake) Authorize(username, password, twofactor string) (*empire.User, error) {
	return &empire.User{Name: "fake", GitHubToken: "token"}, nil
}

type MembershipError struct {
	Organization string
}

func (e *MembershipError) Error() string {
	return fmt.Sprintf("authorization: not a member of %s", e.Organization)
}
