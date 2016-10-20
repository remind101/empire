// Package onelogin implements an auth.Authenticator for OneLogin delegated
// authentication using SAML.
package onelogin

import (
	"fmt"

	"github.com/remind101/empire/pkg/onelogin"
	"github.com/remind101/empire/pkg/saml"
	"github.com/remind101/empire/server/auth"
	samlauth "github.com/remind101/empire/server/auth/saml"
)

// oneloginClient duck types the interface that we need from onelogin.Client.
type oneloginClient interface {
	GenerateSAMLAssertion(options onelogin.GenerateSAMLAssertionOptions) (*onelogin.GenerateSAMLAssertionResponse, error)
	VerifyFactorSAML(options onelogin.VerifyFactorSAMLOptions) (*onelogin.VerifyFactorSAMLResponse, error)
}

type serviceProvider interface {
	ParseSAMLResponse(string, []string) (*saml.Assertion, error)
}

// Authenticator implements the auth.Authenticator interface.
type Authenticator struct {
	// Onelogin SAML app id.
	AppID string

	// Onelogin subdomain.
	Subdomain string

	onelogin oneloginClient

	// ServiceProvider that will be used to verify the SAML response.
	sp serviceProvider
}

// NewAuthenticator returns a new Authenticator instance backed by the given
// client.
func NewAuthenticator(sp *saml.ServiceProvider, clientID, clientSecret string) *Authenticator {
	c := onelogin.New(nil)
	c.ClientID = clientID
	c.ClientSecret = clientSecret

	return &Authenticator{
		onelogin: c,
		sp:       sp,
	}
}

func (a *Authenticator) Authenticate(username, password, otp string) (*auth.Session, error) {
	resp, err := a.onelogin.GenerateSAMLAssertion(onelogin.GenerateSAMLAssertionOptions{
		UsernameOrEmail: username,
		Password:        password,
		AppID:           a.AppID,
		Subdomain:       a.Subdomain,
	})
	if err != nil {
		return nil, handleAuthError(err)
	}

	var samlResponse string
	switch v := resp.Data.(type) {
	case []*onelogin.GenerateSAMLAssertionMFAData:
		// MFA is required, so if we don't have an otp, we can't
		// continue.
		if otp == "" {
			return nil, auth.ErrTwoFactor
		}

		if len(v) != 1 {
			return nil, fmt.Errorf("onelogin: unexpected number of GenerateSAMLAssertionMFAData: %d", len(v))
		}

		data := v[0]

		if len(data.Devices) <= 0 {
			return nil, fmt.Errorf("onelogin: MFA is required, but user has no devices")
		}

		// TODO: There could be multiple devices here. What do we do in
		// that case?
		device := data.Devices[0]

		resp, err := a.onelogin.VerifyFactorSAML(onelogin.VerifyFactorSAMLOptions{
			AppID:      a.AppID,
			DeviceID:   fmt.Sprintf("%d", device.DeviceID),
			StateToken: data.StateToken,
			OTPToken:   &otp,
		})
		if err != nil {
			return nil, handleAuthError(err)
		}

		samlResponse = resp.Data
	case string:
		samlResponse = v
	}

	assertion, err := a.sp.ParseSAMLResponse(samlResponse, []string{""})
	if err != nil {
		return nil, err
	}

	return samlauth.SessionFromAssertion(assertion), nil
}

// handleAuthError handles a onelogin API error. If the error is a 401, then we
// return auth.Forbidden.
func handleAuthError(err error) error {
	if err, ok := err.(*onelogin.Error); ok {
		switch err.Status.Code {
		case 401:
			return auth.ErrForbidden
		}
	}
	return err
}
