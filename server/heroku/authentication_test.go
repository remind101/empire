package heroku

import (
	"net/http"
	"testing"

	"github.com/remind101/empire"
	"github.com/remind101/empire/server/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthentication_UsernamePassword(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Server{
		Authenticator: a,
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(&empire.User{}, nil)

	_, err := m.Authenticate(req)
	assert.NoError(t, err)
}

func TestAuthentication_UsernamePasswordWithOTP(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Server{
		Authenticator: a,
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")
	req.Header.Set("Heroku-Two-Factor-Code", "otp")

	a.On("Authenticate", "username", "password", "otp").Return(&empire.User{}, nil)

	_, err := m.Authenticate(req)
	assert.NoError(t, err)
}

func TestAuthentication_ErrTwoFactor(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Server{
		Authenticator: a,
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(nil, auth.ErrTwoFactor)

	_, err := m.Authenticate(req)
	assert.Equal(t, ErrTwoFactor, err)
}

func TestAuthentication_ErrForbidden(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Server{
		Authenticator: a,
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(nil, auth.ErrForbidden)

	_, err := m.Authenticate(req)
	assert.Equal(t, ErrUnauthorized, err) // TODO: ErrForbidden?
}

func TestAuthentication_UnauthorizedError(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Server{
		Authenticator: a,
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(nil, &auth.UnauthorizedError{
		Reason: "Because you smell",
	})

	_, err := m.Authenticate(req)
	assert.Equal(t, &ErrorResource{
		Status:  http.StatusUnauthorized,
		ID:      "unauthorized",
		Message: "Because you smell",
	}, err)
}

type mockAuthenticator struct {
	mock.Mock
}

func (m *mockAuthenticator) Authenticate(username, password, otp string) (*empire.User, error) {
	args := m.Called(username, password, otp)
	user := args.Get(0)
	if user != nil {
		return user.(*empire.User), args.Error(1)
	}
	return nil, args.Error(1)

}

// ensureUserInContext returns and http.Handler that raises an error if the
// user isn't set in the context.
func ensureUserInContext(t testing.TB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		UserFromContext(r.Context()) // Panics if user is not set.
	})
}
