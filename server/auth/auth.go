// Package auth contains types for authenticating and authorizing requests.
package auth

import (
	"context"
	"errors"

	"github.com/remind101/empire"
	"github.com/remind101/empire/acl"
)

var (
	// ErrForbidden can be returned from Authenticator implementations when
	// the user provides invalid credentials.
	ErrForbidden = errors.New("auth: forbidden")

	// ErrTwoFactor can be returned by an Authenticator implementation when
	// a two factor code is either invalid or required.
	ErrTwoFactor = errors.New("auth: two factor code required or invalid")
)

// Policy is something that returns a set of acl policy to use for the given
// user.
type Policy interface {
	Policy(*empire.User) (acl.Policy, error)
}

type policyFunc func(*empire.User) (acl.Policy, error)

func (f policyFunc) Policy(user *empire.User) (acl.Policy, error) {
	return f(user)
}

// Returns a Policy implementation that will always return the given policy.
func StaticPolicy(policy acl.Policy) Policy {
	return policyFunc(func(user *empire.User) (acl.Policy, error) {
		return policy, nil
	})
}

// Auth provides a simple wrapper around, authenticating the user,
// pre-authorizing the request, then embedding a set of ACL policy to
// authorize the action.
type Auth struct {
	Authenticator Authenticator
	Authorizer    Authorizer
	Policy        Policy
}

// Authenticate authenticates the request, and returns a new context.Context
// with the user embedded. The user can be retrieved with UserFromContext.
func (a *Auth) Authenticate(ctx context.Context, username, password, otp string) (context.Context, error) {
	user, err := a.Authenticator.Authenticate(username, password, otp)
	if err != nil {
		return ctx, err
	}

	ctx = WithUser(ctx, user)

	if a.Authorizer != nil {
		if err := a.Authorizer.Authorize(user); err != nil {
			return ctx, err
		}
	}

	if a.Policy != nil {
		policy, err := a.Policy.Policy(user)
		if err != nil {
			return ctx, err
		}

		ctx = acl.WithPolicy(ctx, policy)
	}

	return ctx, nil
}

// UnauthorizedError can be returned from Authorizer implementations when the
// user is not authorized to perform an action.
type UnauthorizedError struct {
	// A user friendly message for why the user isn't authorized.
	Reason string
}

func (e *UnauthorizedError) Error() string {
	return e.Reason
}

// Authenticator represents something that, given a username, password and OTP
// can authenticate an Empire user.
type Authenticator interface {
	// Authenticate should check the credentials and return the Empire user.
	Authenticate(username, password, twofactor string) (*empire.User, error)
}

// AuthenticatorFunc is a function signature that implements the Authenticator
// interface.
type AuthenticatorFunc func(string, string, string) (*empire.User, error)

// Authenticate calls the AuthenticatorFunc.
func (fn AuthenticatorFunc) Authenticate(username, password, otp string) (*empire.User, error) {
	return fn(username, password, otp)
}

// Authorizer represents something that can perform an authorization check.
type Authorizer interface {
	// Authorize should check that the user has access to perform the
	// action. If not, ErrUnauthorized should be returned.
	Authorize(*empire.User) error
}

type AuthorizerFunc func(*empire.User) error

func (fn AuthorizerFunc) Authorize(user *empire.User) error {
	return fn(user)
}

// StaticAuthenticator returns an Authenticator that returns the provided user
// when the given credentials are provided.
func StaticAuthenticator(username, password, otp string, user *empire.User) Authenticator {
	return AuthenticatorFunc(func(givenUsername, givenPassword, givenOtp string) (*empire.User, error) {
		if givenUsername != username {
			return nil, ErrForbidden
		}

		if givenPassword != password {
			return nil, ErrForbidden
		}

		if givenOtp != otp {
			return nil, ErrTwoFactor
		}

		return user, nil
	})
}

// Anyone returns an Authenticator that let's anyone in and sets them as the
// given user.
func Anyone(user *empire.User) Authenticator {
	return AuthenticatorFunc(func(username, password, otp string) (*empire.User, error) {
		return user, nil
	})
}

// MultiAuthenticator returns an Authenticator that tries each Authenticator
// until one succeeds or they all fail.
//
// It will proceed to the next authenticator when the error returned is
// ErrForbidden. Any other errors are bubbled up (e.g. ErrTwoFactor).
func MultiAuthenticator(authenticators ...Authenticator) Authenticator {
	return AuthenticatorFunc(func(username, password, otp string) (*empire.User, error) {
		for _, authenticator := range authenticators {
			user, err := authenticator.Authenticate(username, password, otp)

			// No error so we're authenticated.
			if err == nil {
				return user, nil
			}

			// Try the next authenticator.
			if err == ErrForbidden {
				continue
			}

			// Bubble up the error.
			return nil, err
		}

		// None succeeded.
		return nil, ErrForbidden
	})
}

// key used to store context values from within this package.
type key int

const (
	userKey key = iota
)

// WithUser adds a user to the context.Context.
func WithUser(ctx context.Context, u *empire.User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// UserFromContext returns a user from a context.Context if one is present.
func UserFromContext(ctx context.Context) *empire.User {
	u, ok := ctx.Value(userKey).(*empire.User)
	if !ok {
		panic("expected user to be authenticated")
	}
	return u
}
