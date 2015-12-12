package auth

import (
	"testing"

	"github.com/remind101/empire"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStaticAuthenticator(t *testing.T) {
	u := &empire.User{}
	a := StaticAuthenticator("username", "password", "otp", u)

	user, err := a.Authenticate("badusername", "password", "otp")
	assert.Equal(t, ErrForbidden, err)
	assert.Nil(t, user)

	user, err = a.Authenticate("username", "badpassword", "otp")
	assert.Equal(t, ErrForbidden, err)
	assert.Nil(t, user)

	user, err = a.Authenticate("username", "password", "badotp")
	assert.Equal(t, ErrTwoFactor, err)
	assert.Nil(t, user)

	user, err = a.Authenticate("username", "password", "otp")
	assert.NoError(t, err)
	assert.Equal(t, u, user)
}

func TestMultiAuthenticator_First(t *testing.T) {
	u := &empire.User{}
	a1 := new(mockAuthenticator)
	a2 := new(mockAuthenticator)
	a := MultiAuthenticator(a1, a2)

	a1.On("Authenticate", "username", "password", "").Return(&empire.User{}, nil)

	user, err := a.Authenticate("username", "password", "")
	assert.NoError(t, err)
	assert.Equal(t, u, user)

	a1.AssertExpectations(t)
	a2.AssertExpectations(t)
}

func TestMultiAuthenticator_Second(t *testing.T) {
	u := &empire.User{}
	a1 := new(mockAuthenticator)
	a2 := new(mockAuthenticator)
	a := MultiAuthenticator(a1, a2)

	a1.On("Authenticate", "username", "password", "").Return(nil, ErrForbidden)
	a2.On("Authenticate", "username", "password", "").Return(&empire.User{}, nil)

	user, err := a.Authenticate("username", "password", "")
	assert.NoError(t, err)
	assert.Equal(t, u, user)

	a1.AssertExpectations(t)
	a2.AssertExpectations(t)
}

func TestMultiAuthenticator_None(t *testing.T) {
	a1 := new(mockAuthenticator)
	a2 := new(mockAuthenticator)
	a := MultiAuthenticator(a1, a2)

	a1.On("Authenticate", "username", "password", "").Return(nil, ErrForbidden)
	a2.On("Authenticate", "username", "password", "").Return(nil, ErrForbidden)

	user, err := a.Authenticate("username", "password", "")
	assert.Equal(t, ErrForbidden, err)
	assert.Nil(t, user)

	a1.AssertExpectations(t)
	a2.AssertExpectations(t)
}

func TestMultiAuthenticator_ErrTwoFactor(t *testing.T) {
	a1 := new(mockAuthenticator)
	a2 := new(mockAuthenticator)
	a := MultiAuthenticator(a1, a2)

	a1.On("Authenticate", "username", "password", "").Return(nil, ErrTwoFactor)

	user, err := a.Authenticate("username", "password", "")
	assert.Equal(t, ErrTwoFactor, err)
	assert.Nil(t, user)

	a1.AssertExpectations(t)
	a2.AssertExpectations(t)
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
