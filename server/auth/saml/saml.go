package saml

import (
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/saml"
	"github.com/remind101/empire/server/auth"
)

// SessionFromAssertion returns a new auth.Session generated from the SAML
// assertion.
func SessionFromAssertion(assertion *saml.Assertion) *auth.Session {
	login := assertion.Subject.NameID.Value
	user := &empire.User{
		Name: login,
	}

	session := auth.NewSession(user)
	session.ExpiresAt = &assertion.AuthnStatement.SessionNotOnOrAfter
	return session
}
