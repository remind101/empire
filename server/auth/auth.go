// Package auth contains types for authenticating and authorizing requests.
package auth

import (
	"errors"

	"github.com/remind101/empire"
)

var (
	// ErrForbidden can be returned from Authenticator implementations when
	// the user provides invalid credentials.
	ErrForbidden = errors.New("auth: forbidden")

	// ErrTwoFactor can be returned by an Authenticator implementation when
	// a two factor code is either invalid or required.
	ErrTwoFactor = errors.New("auth: two factor code required or invalid")
)

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

// WithAuthorization wraps an Authenticator to also perform an Authorization after
// to user is successfully authenticated.
func WithAuthorization(authenticator Authenticator, authorizer Authorizer) Authenticator {
	return AuthenticatorFunc(func(username, password, otp string) (*empire.User, error) {
		user, err := authenticator.Authenticate(username, password, otp)
		if err != nil {
			return user, err
		}

		if err := authorizer.Authorize(user); err != nil {
			return user, err
		}

		return user, nil
	})
}
