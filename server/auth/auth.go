// Package auth contains types for authenticating and authorizing requests.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/remind101/empire"
)

// Some common names for strategies.
const (
	StrategyUsernamePassword = "UsernamePassword"
	StrategyAccessToken      = "AccessToken"
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

// Auth provides a simple wrapper around, authenticating the user,
// pre-authorizing the request, then embedding a set of ACL policy to
// authorize the action.
type Auth struct {
	Strategies Strategies
	Authorizer Authorizer
}

// Strategy wraps an authenticator with a name.
type Strategy struct {
	Authenticator

	// The name of this strategy.
	Name string

	// When true, disables using this strategy by default, unless the
	// strategy is explicitly requested.
	Disabled bool
}

// Strategies wraps a slice of *Strategy with helpers for authenticating with a
// specific strategy.
type Strategies []*Strategy

// AuthenticatorFor builds an Authenticator using the given strategies (by
// name). If no strategies are provided, all strategies will be used. If a
// strategy is not found, a fake strategy will be returned that will return an
// error when used.
func (s Strategies) AuthenticatorFor(strategies ...string) Authenticator {
	var authenticators []Authenticator
	if len(strategies) > 0 {
		for _, name := range strategies {
			strategy := s.strategy(name)
			if strategy == nil {
				panic(fmt.Errorf("unknown strategy: %s", name))
			}
			authenticators = append(authenticators, strategy)
		}
	} else {
		for _, strategy := range s {
			if !strategy.Disabled {
				authenticators = append(authenticators, strategy)
			}
		}
	}
	return MultiAuthenticator(authenticators...)
}

func (s Strategies) strategy(name string) *Strategy {
	for _, strategy := range s {
		if strategy.Name == name {
			return strategy
		}
	}
	return nil
}

func (a *Auth) copy() *Auth {
	return &Auth{
		Strategies: a.Strategies[:],
		Authorizer: a.Authorizer,
	}
}

// AddAuthenticator returns a shallow copy of the Auth object with the given
// authentication method added.
func (a *Auth) PrependAuthenticator(name string, authenticator Authenticator) *Auth {
	c := a.copy()
	strategy := &Strategy{
		Name:          name,
		Authenticator: authenticator,
	}
	c.Strategies = append([]*Strategy{strategy}, c.Strategies...)
	return c
}

// Authenticate authenticates the request using the named strategy, and returns
// a new context.Context with the user embedded. The user can be retrieved with
// UserFromContext.
func (a *Auth) Authenticate(ctx context.Context, username, password, otp string, strategies ...string) (context.Context, error) {
	// Default to using all strategies to authenticate.
	authenticator := a.Strategies.AuthenticatorFor(strategies...)
	return a.authenticate(ctx, authenticator, username, password, otp)
}

func (a *Auth) authenticate(ctx context.Context, authenticator Authenticator, username, password, otp string) (context.Context, error) {
	session, err := authenticator.Authenticate(username, password, otp)
	if err != nil {
		return ctx, err
	}

	ctx = WithSession(ctx, session)

	if a.Authorizer != nil {
		if err := a.Authorizer.Authorize(session.User); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

// Session represents an authenticated Session.
type Session struct {
	// The authenticated User.
	User *empire.User

	// When this Session will expire. The zero value means no expiration.
	ExpiresAt *time.Time
}

// NewSession returns a new Session for the user.
func NewSession(user *empire.User) *Session {
	return &Session{User: user}
}

// Authenticator represents something that, given a username, password and OTP
// can authenticate an Empire user.
type Authenticator interface {
	// Authenticate should check the credentials and return a login Session.
	Authenticate(username, password, twofactor string) (*Session, error)
}

// AuthenticatorFunc is a function signature that implements the Authenticator
// interface.
type AuthenticatorFunc func(string, string, string) (*Session, error)

// Authenticate calls the AuthenticatorFunc.
func (fn AuthenticatorFunc) Authenticate(username, password, otp string) (*Session, error) {
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
	return AuthenticatorFunc(func(givenUsername, givenPassword, givenOtp string) (*Session, error) {
		if givenUsername != username {
			return nil, ErrForbidden
		}

		if givenPassword != password {
			return nil, ErrForbidden
		}

		if givenOtp != otp {
			return nil, ErrTwoFactor
		}

		return NewSession(user), nil
	})
}

// Anyone returns an Authenticator that let's anyone in and sets them as the
// given user.
func Anyone(user *empire.User) Authenticator {
	return AuthenticatorFunc(func(username, password, otp string) (*Session, error) {
		return NewSession(user), nil
	})
}

// MultiAuthenticator returns an Authenticator that tries each Authenticator
// until one succeeds or they all fail.
//
// It will proceed to the next authenticator when the error returned is
// ErrForbidden. Any other errors are bubbled up (e.g. ErrTwoFactor).
func MultiAuthenticator(authenticators ...Authenticator) Authenticator {
	return AuthenticatorFunc(func(username, password, otp string) (*Session, error) {
		for _, authenticator := range authenticators {
			session, err := authenticator.Authenticate(username, password, otp)

			// No error so we're authenticated.
			if err == nil {
				return session, nil
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
	sessionKey key = iota
)

// WithSession embeds the authentication Session in the context.Context.
func WithSession(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, sessionKey, session)
}

// UserFromContext returns a user from a context.Context if one is present.
func UserFromContext(ctx context.Context) *empire.User {
	session := SessionFromContext(ctx)
	return session.User
}

// SessionFromContext returns the embedded Session in the context.Context.
func SessionFromContext(ctx context.Context) *Session {
	session, ok := ctx.Value(sessionKey).(*Session)
	if !ok {
		panic("expected user to be authenticated")
	}
	return session
}
