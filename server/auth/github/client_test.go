package github

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"golang.org/x/oauth2"
)

// oauthConfig is fake oauth config to use in tests.
var oauthConfig = &oauth2.Config{
	ClientID:     "client_id",
	ClientSecret: "client_secret",
	Scopes:       []string{"scope"},
}

func TestClient_CreateAuthorization(t *testing.T) {
	h := new(mockHTTPClient)
	c := &Client{
		Config: oauthConfig,
		client: h,
	}

	req, _ := http.NewRequest("POST", "https://api.github.com/authorizations", bytes.NewBufferString(`{"scopes":["scope"],"client_id":"client_id","client_secret":"client_secret"}`+"\n"))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")

	h.On("Do", req).Return(&http.Response{
		Request:    req,
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"token":"access_token"}`)),
	}, nil)

	auth, err := c.CreateAuthorization(CreateAuthorizationOptions{
		Username: "username",
		Password: "password",
	})
	assert.NoError(t, err)
	assert.Equal(t, "access_token", auth.Token)

	h.AssertExpectations(t)
}

func TestClient_CreateAuthorization_RequiresOTP(t *testing.T) {
	h := new(mockHTTPClient)
	c := &Client{
		Config: oauthConfig,
		client: h,
	}

	req, _ := http.NewRequest("POST", "https://api.github.com/authorizations", bytes.NewBufferString(`{"scopes":["scope"],"client_id":"client_id","client_secret":"client_secret"}`+"\n"))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")

	headers := make(http.Header)
	headers.Set("X-GitHub-OTP", "required; sms")
	h.On("Do", req).Return(&http.Response{
		Request:    req,
		Header:     headers,
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"token":"access_token"}`)),
	}, nil)

	auth, err := c.CreateAuthorization(CreateAuthorizationOptions{
		Username: "username",
		Password: "password",
	})
	assert.Equal(t, errTwoFactor, err)
	assert.Nil(t, auth)

	h.AssertExpectations(t)
}

func TestClient_CreateAuthorization_WithOTP(t *testing.T) {
	h := new(mockHTTPClient)
	c := &Client{
		Config: oauthConfig,
		client: h,
	}

	req, _ := http.NewRequest("POST", "https://api.github.com/authorizations", bytes.NewBufferString(`{"scopes":["scope"],"client_id":"client_id","client_secret":"client_secret"}`+"\n"))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")
	req.Header.Set("X-Github-Otp", "otp")

	h.On("Do", req).Return(&http.Response{
		Request:    req,
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"token":"access_token"}`)),
	}, nil)

	auth, err := c.CreateAuthorization(CreateAuthorizationOptions{
		Username: "username",
		Password: "password",
		OTP:      "otp",
	})
	assert.NoError(t, err)
	assert.Equal(t, "access_token", auth.Token)

	h.AssertExpectations(t)
}

func TestClient_CreateAuthorization_Unauthorized(t *testing.T) {
	h := new(mockHTTPClient)
	c := &Client{
		Config: oauthConfig,
		client: h,
	}

	req, _ := http.NewRequest("POST", "https://api.github.com/authorizations", bytes.NewBufferString(`{"scopes":["scope"],"client_id":"client_id","client_secret":"client_secret"}`+"\n"))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")

	h.On("Do", req).Return(&http.Response{
		Request:    req,
		StatusCode: http.StatusUnauthorized,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{}`)),
	}, nil)

	auth, err := c.CreateAuthorization(CreateAuthorizationOptions{
		Username: "username",
		Password: "password",
	})
	assert.Equal(t, errUnauthorized, err)
	assert.Nil(t, auth)

	h.AssertExpectations(t)
}

func TestClient_CreateAuthorization_Error(t *testing.T) {
	h := new(mockHTTPClient)
	c := &Client{
		Config: oauthConfig,
		client: h,
	}

	req, _ := http.NewRequest("POST", "https://api.github.com/authorizations", bytes.NewBufferString(`{"scopes":["scope"],"client_id":"client_id","client_secret":"client_secret"}`+"\n"))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", "Basic dXNlcm5hbWU6cGFzc3dvcmQ=")

	h.On("Do", req).Return(&http.Response{
		Request:    req,
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"message":"our SMS provider doesn't deliver to your area"}`)),
	}, nil)

	auth, err := c.CreateAuthorization(CreateAuthorizationOptions{
		Username: "username",
		Password: "password",
	})
	assert.EqualError(t, err, "github: our SMS provider doesn't deliver to your area")
	assert.Nil(t, auth)

	h.AssertExpectations(t)
}

func TestClient_GetUser(t *testing.T) {
	h := new(mockHTTPClient)
	c := &Client{
		Config: oauthConfig,
		client: h,
	}

	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.SetBasicAuth("access_token", "x-oauth-basic")

	h.On("Do", req).Return(&http.Response{
		Request:    req,
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"login":"ejholmes"}`)),
	}, nil)

	user, err := c.GetUser("access_token")
	assert.NoError(t, err)
	assert.Equal(t, "ejholmes", user.Login)

	h.AssertExpectations(t)
}

func TestClient_GetUser_Error(t *testing.T) {
	h := new(mockHTTPClient)
	c := &Client{
		Config: oauthConfig,
		client: h,
		backoff: func(try int) time.Duration {
			return 0
		},
	}

	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.SetBasicAuth("access_token", "x-oauth-basic")

	h.On("Do", req).Return(&http.Response{
		Request:    req,
		StatusCode: http.StatusNotFound,
		Body:       ioutil.NopCloser(bytes.NewBufferString(`{"message":"not found"}`)),
	}, nil).Times(3)

	_, err := c.GetUser("access_token")
	assert.Error(t, err)

	h.AssertExpectations(t)
}

func TestClient_IsOrganizationMember(t *testing.T) {
	tests := []struct {
		status int
		member bool
	}{
		{200, true},
		{204, true},
		{404, false},
	}

	for _, tt := range tests {
		h := new(mockHTTPClient)
		c := &Client{
			Config: oauthConfig,
			client: h,
		}

		req, _ := http.NewRequest("HEAD", "https://api.github.com/user/memberships/orgs/remind101", nil)
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.SetBasicAuth("access_token", "x-oauth-basic")

		h.On("Do", req).Return(&http.Response{
			Request:    req,
			StatusCode: tt.status,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`{"login":"ejholmes"}`)),
		}, nil)

		ok, err := c.IsOrganizationMember("remind101", "access_token")
		assert.NoError(t, err)
		assert.Equal(t, tt.member, ok)

		h.AssertExpectations(t)
	}
}

func TestClient_IsTeamMember(t *testing.T) {
	tests := []struct {
		status int
		state  string
		member bool
	}{
		{200, "active", true},
		{200, "pending", false},
		{404, "", false},
	}

	for _, tt := range tests {
		h := new(mockHTTPClient)
		c := &Client{
			Config: oauthConfig,
			client: h,
		}

		req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.SetBasicAuth("access_token", "x-oauth-basic")

		h.On("Do", req).Return(&http.Response{
			Request:    req,
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`{"login":"ejholmes"}`)),
		}, nil)

		req, _ = http.NewRequest("GET", "https://api.github.com/teams/123/memberships/ejholmes", nil)
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.SetBasicAuth("access_token", "x-oauth-basic")

		h.On("Do", req).Return(&http.Response{
			Request:    req,
			StatusCode: tt.status,
			Body:       ioutil.NopCloser(bytes.NewBufferString(fmt.Sprintf("{\"state\": \"%s\"}", tt.state))),
		}, nil)

		ok, err := c.IsTeamMember("123", "access_token")
		assert.NoError(t, err)
		assert.Equal(t, tt.member, ok)

		h.AssertExpectations(t)
	}
}

type mockHTTPClient struct {
	mock.Mock
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}
