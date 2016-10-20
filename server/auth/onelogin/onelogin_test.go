package onelogin

import (
	"testing"
	"time"

	"github.com/remind101/empire/pkg/onelogin"
	"github.com/remind101/empire/pkg/saml"
	"github.com/remind101/empire/server/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthenticator(t *testing.T) {
	sp := new(mockServiceProvider)
	c := new(mockOnelogin)
	a := &Authenticator{
		AppID:     "1234",
		Subdomain: "acme",
		sp:        sp,
		onelogin:  c,
	}

	c.On("GenerateSAMLAssertion", onelogin.GenerateSAMLAssertionOptions{
		UsernameOrEmail: "username",
		Password:        "password",
		AppID:           "1234",
		Subdomain:       "acme",
	}).Return(&onelogin.GenerateSAMLAssertionResponse{
		Data: `SAMLResponse`,
	}, nil)

	session, err := a.Authenticate("username", "password", "")
	assert.NoError(t, err)
	assert.Equal(t, "ejholmes", session.User.Name)
	assert.NotNil(t, session.ExpiresAt)
}

func TestAuthenticator_BadUsernamePassword(t *testing.T) {
	sp := new(mockServiceProvider)
	c := new(mockOnelogin)
	a := &Authenticator{
		AppID:     "1234",
		Subdomain: "acme",
		sp:        sp,
		onelogin:  c,
	}

	var meta onelogin.ResponseMeta
	meta.Status.Code = 401
	c.On("GenerateSAMLAssertion", onelogin.GenerateSAMLAssertionOptions{
		UsernameOrEmail: "username",
		Password:        "password",
		AppID:           "1234",
		Subdomain:       "acme",
	}).Return(nil, &onelogin.Error{
		ResponseMeta: meta,
	})

	_, err := a.Authenticate("username", "password", "")
	assert.Equal(t, auth.ErrForbidden, err)
}

func TestAuthenticator_WithOTP(t *testing.T) {
	sp := new(mockServiceProvider)
	c := new(mockOnelogin)
	a := &Authenticator{
		AppID:     "1234",
		Subdomain: "acme",
		sp:        sp,
		onelogin:  c,
	}

	c.On("GenerateSAMLAssertion", onelogin.GenerateSAMLAssertionOptions{
		UsernameOrEmail: "username",
		Password:        "password",
		AppID:           "1234",
		Subdomain:       "acme",
	}).Return(&onelogin.GenerateSAMLAssertionResponse{
		Data: []*onelogin.GenerateSAMLAssertionMFAData{
			{
				StateToken: "5xxx604x8xx9x694xx860173xxx3x78x3x870x56",
				Devices: []*onelogin.Device{
					{
						DeviceID:   666666,
						DeviceType: "Duo",
					},
				},
			},
		},
	}, nil)

	otp := "otp"
	c.On("VerifyFactorSAML", onelogin.VerifyFactorSAMLOptions{
		AppID:      "1234",
		StateToken: "5xxx604x8xx9x694xx860173xxx3x78x3x870x56",
		DeviceID:   "666666",
		OTPToken:   &otp,
	}).Return(&onelogin.VerifyFactorSAMLResponse{
		Data: `<SAMLResponse>`,
	}, nil)

	session, err := a.Authenticate("username", "password", "otp")
	assert.NoError(t, err)
	assert.Equal(t, "ejholmes", session.User.Name)
	assert.NotNil(t, session.ExpiresAt)
}

func TestAuthenticator_BadOTP(t *testing.T) {
	sp := new(mockServiceProvider)
	c := new(mockOnelogin)
	a := &Authenticator{
		AppID:     "1234",
		Subdomain: "acme",
		sp:        sp,
		onelogin:  c,
	}

	c.On("GenerateSAMLAssertion", onelogin.GenerateSAMLAssertionOptions{
		UsernameOrEmail: "username",
		Password:        "password",
		AppID:           "1234",
		Subdomain:       "acme",
	}).Return(&onelogin.GenerateSAMLAssertionResponse{
		Data: []*onelogin.GenerateSAMLAssertionMFAData{
			{
				StateToken: "5xxx604x8xx9x694xx860173xxx3x78x3x870x56",
				Devices: []*onelogin.Device{
					{
						DeviceID:   666666,
						DeviceType: "Duo",
					},
				},
			},
		},
	}, nil)

	otp := "otp"
	var meta onelogin.ResponseMeta
	meta.Status.Code = 401
	meta.Status.Type = "Unauthorized"
	meta.Status.Message = "Failed authentication with this factor"
	c.On("VerifyFactorSAML", onelogin.VerifyFactorSAMLOptions{
		AppID:      "1234",
		StateToken: "5xxx604x8xx9x694xx860173xxx3x78x3x870x56",
		DeviceID:   "666666",
		OTPToken:   &otp,
	}).Return(nil, &onelogin.Error{
		ResponseMeta: meta,
	})

	_, err := a.Authenticate("username", "password", "otp")
	assert.Equal(t, auth.ErrForbidden, err)
}

func TestAuthenticator_MissingOTP(t *testing.T) {
	sp := new(mockServiceProvider)
	c := new(mockOnelogin)
	a := &Authenticator{
		AppID:     "1234",
		Subdomain: "acme",
		sp:        sp,
		onelogin:  c,
	}

	c.On("GenerateSAMLAssertion", onelogin.GenerateSAMLAssertionOptions{
		UsernameOrEmail: "username",
		Password:        "password",
		AppID:           "1234",
		Subdomain:       "acme",
	}).Return(&onelogin.GenerateSAMLAssertionResponse{
		Data: []*onelogin.GenerateSAMLAssertionMFAData{
			{
				StateToken: "5xxx604x8xx9x694xx860173xxx3x78x3x870x56",
				Devices: []*onelogin.Device{
					{
						DeviceID:   666666,
						DeviceType: "Duo",
					},
				},
			},
		},
	}, nil)

	_, err := a.Authenticate("username", "password", "")
	assert.Equal(t, auth.ErrTwoFactor, err)
}

type mockOnelogin struct {
	mock.Mock
}

func (m *mockOnelogin) GenerateSAMLAssertion(options onelogin.GenerateSAMLAssertionOptions) (*onelogin.GenerateSAMLAssertionResponse, error) {
	args := m.Called(options)
	var resp *onelogin.GenerateSAMLAssertionResponse
	if v := args.Get(0); v != nil {
		resp = v.(*onelogin.GenerateSAMLAssertionResponse)
	}
	return resp, args.Error(1)
}

func (m *mockOnelogin) VerifyFactorSAML(options onelogin.VerifyFactorSAMLOptions) (*onelogin.VerifyFactorSAMLResponse, error) {
	args := m.Called(options)
	var resp *onelogin.VerifyFactorSAMLResponse
	if v := args.Get(0); v != nil {
		resp = v.(*onelogin.VerifyFactorSAMLResponse)
	}
	return resp, args.Error(1)
}

type mockServiceProvider struct{}

func (m *mockServiceProvider) ParseSAMLResponse(samlResponse string, possibleRequestIds []string) (*saml.Assertion, error) {
	return &saml.Assertion{
		Subject: &saml.Subject{
			NameID: &saml.NameID{
				Value: "ejholmes",
			},
		},
		AuthnStatement: &saml.AuthnStatement{
			SessionNotOnOrAfter: time.Now().Add(24 * time.Hour),
		},
	}, nil
}
