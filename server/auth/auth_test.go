package auth

import (
	"testing"
	"time"

	"github.com/remind101/empire"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAuthenticator struct {
	mock.Mock
}

func (m *mockAuthenticator) Authenticate(username, password, otp string) (*Session, error) {
	args := m.Called(username, password, otp)
	session := args.Get(0)
	if session != nil {
		return session.(*Session), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestStaticAuthenticator(t *testing.T) {
	u := &empire.User{}
	a := StaticAuthenticator("username", "password", "otp", u)

	session, err := a.Authenticate("badusername", "password", "otp")
	assert.Equal(t, ErrForbidden, err)
	assert.Nil(t, session)

	session, err = a.Authenticate("username", "badpassword", "otp")
	assert.Equal(t, ErrForbidden, err)
	assert.Nil(t, session)

	session, err = a.Authenticate("username", "password", "badotp")
	assert.Equal(t, ErrTwoFactor, err)
	assert.Nil(t, session)

	session, err = a.Authenticate("username", "password", "otp")
	assert.NoError(t, err)
	assert.Equal(t, NewSession(u), session)
}

func TestMultiAuthenticator_First(t *testing.T) {
	u := &empire.User{}
	s := NewSession(u)
	a1 := new(mockAuthenticator)
	a2 := new(mockAuthenticator)
	a := MultiAuthenticator(a1, a2)

	a1.On("Authenticate", "username", "password", "").Return(s, nil)

	session, err := a.Authenticate("username", "password", "")
	assert.NoError(t, err)
	assert.Equal(t, s, session)

	a1.AssertExpectations(t)
	a2.AssertExpectations(t)
}

func TestMultiAuthenticator_Second(t *testing.T) {
	u := &empire.User{}
	s := NewSession(u)
	a1 := new(mockAuthenticator)
	a2 := new(mockAuthenticator)
	a := MultiAuthenticator(a1, a2)

	a1.On("Authenticate", "username", "password", "").Return(nil, ErrForbidden)
	a2.On("Authenticate", "username", "password", "").Return(s, nil)

	session, err := a.Authenticate("username", "password", "")
	assert.NoError(t, err)
	assert.Equal(t, s, session)

	a1.AssertExpectations(t)
	a2.AssertExpectations(t)
}

func TestMultiAuthenticator_None(t *testing.T) {
	a1 := new(mockAuthenticator)
	a2 := new(mockAuthenticator)
	a := MultiAuthenticator(a1, a2)

	a1.On("Authenticate", "username", "password", "").Return(nil, ErrForbidden)
	a2.On("Authenticate", "username", "password", "").Return(nil, ErrForbidden)

	session, err := a.Authenticate("username", "password", "")
	assert.Equal(t, ErrForbidden, err)
	assert.Nil(t, session)

	a1.AssertExpectations(t)
	a2.AssertExpectations(t)
}

func TestMultiAuthenticator_ErrTwoFactor(t *testing.T) {
	a1 := new(mockAuthenticator)
	a2 := new(mockAuthenticator)
	a := MultiAuthenticator(a1, a2)

	a1.On("Authenticate", "username", "password", "").Return(nil, ErrTwoFactor)

	session, err := a.Authenticate("username", "password", "")
	assert.Equal(t, ErrTwoFactor, err)
	assert.Nil(t, session)

	a1.AssertExpectations(t)
	a2.AssertExpectations(t)
}

func TestWithMaxSessionDuration(t *testing.T) {
	t1 := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	m := new(mockAuthenticator)
	a := WithMaxSessionDuration(m, func() time.Time { return t1 })

	m.On("Authenticate", "username", "password", "").Return(&Session{}, nil)

	session, err := a.Authenticate("username", "password", "")
	assert.NoError(t, err)
	assert.Equal(t, t1, *session.ExpiresAt)

	m.AssertExpectations(t)
}

func TestWithMaxSessionDuration_WithShorterExpiresAt(t *testing.T) {
	s1 := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	m := new(mockAuthenticator)
	a := WithMaxSessionDuration(m, func() time.Time { return s1.Add(24 * time.Hour) })

	exp := s1.Add(12 * time.Hour)
	m.On("Authenticate", "username", "password", "").Return(&Session{
		ExpiresAt: &exp,
	}, nil)

	session, err := a.Authenticate("username", "password", "")
	assert.NoError(t, err)
	assert.Equal(t, exp, *session.ExpiresAt)

	m.AssertExpectations(t)
}

func TestWithMaxSessionDuration_WithLongerExpiresAt(t *testing.T) {
	s1 := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	m := new(mockAuthenticator)
	a := WithMaxSessionDuration(m, func() time.Time { return s1.Add(12 * time.Hour) })

	exp := s1.Add(24 * time.Hour)
	m.On("Authenticate", "username", "password", "").Return(&Session{
		ExpiresAt: &exp,
	}, nil)

	session, err := a.Authenticate("username", "password", "")
	assert.NoError(t, err)
	assert.Equal(t, s1.Add(12*time.Hour), *session.ExpiresAt)

	m.AssertExpectations(t)
}
