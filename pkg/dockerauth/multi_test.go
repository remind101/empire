package dockerauth

import (
	"testing"
	"fmt"

	"github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMultiAuthProvider_AuthConfiguration_FirstMatch(t *testing.T) {
	mockProvider1 := new(mockAuthProvider)
	mockProvider1.On("AuthConfiguration", "foobar").Return(&docker.AuthConfiguration{Username: "foo"}, nil)
	mockProvider2 := new(mockAuthProvider)
	provider := NewMultiAuthProvider(mockProvider1, mockProvider2)

	authConf, err := provider.AuthConfiguration("foobar")
	assert.NoError(t, err)
	assert.Equal(t, "foo", authConf.Username)
	mockProvider1.AssertCalled(t, "AuthConfiguration", "foobar")
	mockProvider1.AssertNumberOfCalls(t, "AuthConfiguration", 1)
	mockProvider2.AssertNumberOfCalls(t, "AuthConfiguration", 0)
}

func TestMultiAuthProvider_AuthConfiguration_SecondMatch(t *testing.T) {
	mockProvider1 := new(mockAuthProvider)
	mockProvider1.On("AuthConfiguration", "foobar").Return(nil, nil)
	mockProvider2 := new(mockAuthProvider)
	mockProvider2.On("AuthConfiguration", "foobar").Return(&docker.AuthConfiguration{Username: "foo"}, nil)
	provider := NewMultiAuthProvider(mockProvider1, mockProvider2)

	authConf, err := provider.AuthConfiguration("foobar")
	assert.NoError(t, err)
	assert.Equal(t, "foo", authConf.Username)
	mockProvider1.AssertCalled(t, "AuthConfiguration", "foobar")
	mockProvider1.AssertNumberOfCalls(t, "AuthConfiguration", 1)
	mockProvider2.AssertCalled(t, "AuthConfiguration", "foobar")
	mockProvider2.AssertNumberOfCalls(t, "AuthConfiguration", 1)
}

func TestMultiAuthProvider_AuthConfiguration_FirstErr(t *testing.T) {
	mockProvider1 := new(mockAuthProvider)
	mockProvider1.On("AuthConfiguration", "foobar").Return(nil, fmt.Errorf("Some error"))
	mockProvider2 := new(mockAuthProvider)
	mockProvider2.On("AuthConfiguration", "foobar").Return(nil, nil)
	provider := NewMultiAuthProvider(mockProvider1, mockProvider2)

	_, err := provider.AuthConfiguration("foobar")
	assert.EqualError(t, err, "Some error")
	mockProvider1.AssertCalled(t, "AuthConfiguration", "foobar")
	mockProvider1.AssertNumberOfCalls(t, "AuthConfiguration", 1)
	mockProvider2.AssertNumberOfCalls(t, "AuthConfiguration", 0)
}

func TestMultiAuthProvider_AuthConfiguration_NoMatch(t *testing.T) {
	mockProvider1 := new(mockAuthProvider)
	mockProvider1.On("AuthConfiguration", "foobar").Return(nil, nil)
	mockProvider2 := new(mockAuthProvider)
	mockProvider2.On("AuthConfiguration", "foobar").Return(nil, nil)
	provider := NewMultiAuthProvider(mockProvider1, mockProvider2)

	authConf, err := provider.AuthConfiguration("foobar")
	assert.NoError(t, err)
	assert.Nil(t, authConf)
	mockProvider1.AssertCalled(t, "AuthConfiguration", "foobar")
	mockProvider1.AssertNumberOfCalls(t, "AuthConfiguration", 1)
	mockProvider2.AssertCalled(t, "AuthConfiguration", "foobar")
	mockProvider2.AssertNumberOfCalls(t, "AuthConfiguration", 1)
}

func TestMultiAuthProvider_AddProvider(t *testing.T) {
	p := NewMultiAuthProvider()
	assert.Equal(t, 0, len(p.providers))

	mockProvider1 := new(mockAuthProvider)
	p.AddProvider(mockProvider1)
	assert.Equal(t, p.providers, []AuthProvider{mockProvider1})

	mockProvider2 := new(mockAuthProvider)
	p.AddProvider(mockProvider2)
	assert.Equal(t, p.providers, []AuthProvider{mockProvider1, mockProvider2})
}

type mockAuthProvider struct {
	mock.Mock
}

func (m *mockAuthProvider) AuthConfiguration(registry string) (*docker.AuthConfiguration, error) {
	ret := m.Called(registry)
	authConf := ret.Get(0)
	if authConf != nil {
		return ret.Get(0).(*docker.AuthConfiguration), ret.Error(1)
	} else {
		return nil, ret.Error(1)
	}
}
