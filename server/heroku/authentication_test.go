package heroku

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remind101/empire"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/pkg/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

func TestAuthentication_UsernamePassword(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Authentication{
		authenticator: a,
		handler:       ensureUserInContext(t),
	}

	ctx := context.Background()
	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(&empire.User{}, nil)

	err := m.ServeHTTPContext(ctx, resp, req)
	assert.NoError(t, err)
}

func TestAuthentication_UsernamePasswordWithOTP(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Authentication{
		authenticator: a,
		handler:       ensureUserInContext(t),
	}

	ctx := context.Background()
	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")
	req.Header.Set("Heroku-Two-Factor-Code", "otp")

	a.On("Authenticate", "username", "password", "otp").Return(&empire.User{}, nil)

	err := m.ServeHTTPContext(ctx, resp, req)
	assert.NoError(t, err)
}

func TestAuthentication_ErrTwoFactor(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Authentication{
		authenticator: a,
	}

	ctx := context.Background()
	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(nil, auth.ErrTwoFactor)

	err := m.ServeHTTPContext(ctx, resp, req)
	assert.Equal(t, ErrTwoFactor, err)
}

func TestAuthentication_ErrForbidden(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Authentication{
		authenticator: a,
	}

	ctx := context.Background()
	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(nil, auth.ErrForbidden)

	err := m.ServeHTTPContext(ctx, resp, req)
	assert.Equal(t, ErrUnauthorized, err) // TODO: ErrForbidden?
}

func TestAuthentication_UnauthorizedError(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Authentication{
		authenticator: a,
	}

	ctx := context.Background()
	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(nil, &auth.UnauthorizedError{
		Reason: "Because you smell",
	})

	err := m.ServeHTTPContext(ctx, resp, req)
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

// ensureUserInContext returns and httpx.Handler that raises an error if the
// user isn't set in the context.
func ensureUserInContext(t testing.TB) httpx.Handler {
	return httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		UserFromContext(ctx) // Panics if user is not set.
		return nil
	})
}
