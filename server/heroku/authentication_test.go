package heroku

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/remind101/empire"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/pkg/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

var testSecret = []byte("secret")
var ctx = context.Background()

func TestServer_AccessTokens(t *testing.T) {
	s := &Server{Secret: testSecret}

	token := &AccessToken{
		User: &empire.User{Name: "ejholmes"},
	}
	_, err := s.AccessTokensCreate(token)
	assert.NoError(t, err)

	token, err = s.AccessTokensFind(token.Token)
	assert.NoError(t, err)
	assert.NotNil(t, token)
	assert.Equal(t, "ejholmes", token.User.Name)

	token, err = s.AccessTokensFind("invalid")
	assert.NoError(t, err)
	assert.Nil(t, token)

	token = &AccessToken{
		User: &empire.User{Name: ""},
	}
	_, err = s.AccessTokensCreate(token)
	assert.Equal(t, empire.ErrUserName, err)
}

func TestServer_Authenticate_UsernamePassword(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Server{Auth: newAuth(a)}

	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(auth.NewSession(&empire.User{}), nil)

	_, err := m.Authenticate(ctx, req)
	assert.NoError(t, err)
}

func TestServer_Authenticate_WithUnknownStrategy(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Server{Auth: newAuth(a)}

	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(&empire.User{}, nil)

	assert.Panics(t, func() {
		m.Authenticate(ctx, req, "mock")
	}, "Calling Authenticate with an unknown strategy should panic")
}

func TestServer_Authenticate_WithStrategy(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Server{Auth: newAuth(a)}

	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(auth.NewSession(&empire.User{}), nil)

	_, err := m.Authenticate(ctx, req)
	assert.NoError(t, err)

	// The provided credentials aren't an access token, so this should
	// return ErrUnauthorized.
	_, err = m.Authenticate(ctx, req, auth.StrategyAccessToken)
	assert.Equal(t, ErrUnauthorized, err)
}

func TestServer_Authenticate_UsernamePasswordWithOTP(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Server{Auth: newAuth(a)}

	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")
	req.Header.Set("Heroku-Two-Factor-Code", "otp")

	a.On("Authenticate", "username", "password", "otp").Return(auth.NewSession(&empire.User{}), nil)

	_, err := m.Authenticate(ctx, req)
	assert.NoError(t, err)
}

func TestServer_Authenticate_ErrTwoFactor(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Server{Auth: newAuth(a)}

	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(nil, auth.ErrTwoFactor)

	_, err := m.Authenticate(ctx, req)
	assert.Equal(t, ErrTwoFactor, err)
}

func TestServer_Authenticate_ErrForbidden(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Server{Auth: newAuth(a)}

	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(nil, auth.ErrForbidden)

	_, err := m.Authenticate(ctx, req)
	assert.Equal(t, ErrUnauthorized, err) // TODO: ErrForbidden?
}

func TestServer_Authenticate_UnauthorizedError(t *testing.T) {
	a := new(mockAuthenticator)
	m := &Server{Auth: newAuth(a)}

	req, _ := http.NewRequest("GET", "/", nil)
	req.SetBasicAuth("username", "password")

	a.On("Authenticate", "username", "password", "").Return(nil, &auth.UnauthorizedError{
		Reason: "Because you smell",
	})

	_, err := m.Authenticate(ctx, req)
	assert.Equal(t, &ErrorResource{
		Status:  http.StatusUnauthorized,
		ID:      "unauthorized",
		Message: "Because you smell",
	}, err)
}

func TestAccessTokenAuthenticator(t *testing.T) {
	u := &empire.User{}
	a := &accessTokenAuthenticator{
		findAccessToken: func(token string) (*AccessToken, error) {
			assert.Equal(t, "token", token)
			return &AccessToken{
				User: u,
			}, nil
		},
	}

	s := auth.NewSession(u)
	session, err := a.Authenticate("", "token", "")
	assert.NoError(t, err)
	assert.Equal(t, s, session)
}

func TestAccessTokenAuthenticator_TokenNotFound(t *testing.T) {
	a := &accessTokenAuthenticator{
		findAccessToken: func(token string) (*AccessToken, error) {
			assert.Equal(t, "token", token)
			return nil, nil
		},
	}

	session, err := a.Authenticate("", "token", "")
	assert.Equal(t, auth.ErrForbidden, err)
	assert.Nil(t, session)
}

func TestAccessTokenAuthenticator_WithExpiresAt(t *testing.T) {
	exp := time.Now().Add(24 * time.Hour)

	u := &empire.User{}
	a := &accessTokenAuthenticator{
		findAccessToken: func(token string) (*AccessToken, error) {
			assert.Equal(t, "token", token)
			return &AccessToken{
				User:      u,
				ExpiresAt: &exp,
			}, nil
		},
	}

	s := auth.NewSession(u)
	s.ExpiresAt = &exp
	session, err := a.Authenticate("", "token", "")
	assert.NoError(t, err)
	assert.Equal(t, s, session)
}

func TestAccessTokensFind(t *testing.T) {
	s := &Server{Secret: testSecret}

	at, err := s.AccessTokensFind("")
	if err != nil {
		t.Logf("err: %v", reflect.TypeOf(err))
		t.Fatal(err)
	}

	if at != nil {
		t.Fatal("Expected access token to be nil")
	}
}

type mockAuthenticator struct {
	mock.Mock
}

func (m *mockAuthenticator) Authenticate(username, password, otp string) (*auth.Session, error) {
	args := m.Called(username, password, otp)
	session := args.Get(0)
	if session != nil {
		return session.(*auth.Session), args.Error(1)
	}
	return nil, args.Error(1)

}

func newAuth(a *mockAuthenticator) *auth.Auth {
	return &auth.Auth{
		Strategies: auth.Strategies{
			{
				Name:          auth.StrategyUsernamePassword,
				Authenticator: a,
			},
		},
	}
}

// ensureUserInContext returns and httpx.Handler that raises an error if the
// user isn't set in the context.
func ensureUserInContext(t testing.TB) httpx.Handler {
	return httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		auth.UserFromContext(ctx) // Panics if user is not set.
		return nil
	})
}
