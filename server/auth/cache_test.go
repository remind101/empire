package auth

import (
	"testing"
	"time"

	"github.com/remind101/empire"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCachedAuthorizer_Cached(t *testing.T) {
	u := &empire.User{Name: "ejholmes"}
	c := new(mockCache)
	a := &cachedAuthorizer{
		cache: c,
	}

	c.On("Get", "ejholmes").Return(nil, true)

	err := a.Authorize(u)
	assert.NoError(t, err)

	c.AssertExpectations(t)
}

func TestCachedAuthorizer_Positive(t *testing.T) {
	u := &empire.User{Name: "ejholmes"}
	c := new(mockCache)
	m := new(mockAuthorizer)
	a := &cachedAuthorizer{
		Authorizer: m,
		cache:      c,
	}

	m.On("Authorize", u).Return(nil)
	c.On("Get", "ejholmes").Return(nil, false)
	c.On("Set", "ejholmes", true, time.Duration(0))

	err := a.Authorize(u)
	assert.NoError(t, err)

	c.AssertExpectations(t)
	m.AssertExpectations(t)
}

func TestCachedAuthorizer_Negative(t *testing.T) {
	u := &empire.User{Name: "ejholmes"}
	c := new(mockCache)
	m := new(mockAuthorizer)
	a := &cachedAuthorizer{
		Authorizer: m,
		cache:      c,
	}

	errUnauthed := &UnauthorizedError{Reason: "Smells"}
	m.On("Authorize", u).Return(errUnauthed)
	c.On("Get", "ejholmes").Return(nil, false)

	err := a.Authorize(u)
	assert.Equal(t, errUnauthed, err)

	c.AssertExpectations(t)
	m.AssertExpectations(t)
}

type mockCache struct {
	mock.Mock
}

func (m *mockCache) Set(k string, x interface{}, d time.Duration) {
	m.Called(k, x, d)
}

func (m *mockCache) Get(k string) (interface{}, bool) {
	args := m.Called(k)
	return args.Get(0), args.Bool(1)
}

type mockAuthorizer struct {
	mock.Mock
}

func (m *mockAuthorizer) Authorize(user *empire.User) error {
	args := m.Called(user)
	return args.Error(0)
}
