// Pacakge onelogin implements a client for the onelogin API.
//
// https://developers.onelogin.com/api-docs/1/getting-started/dev-overview
package onelogin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Default OneLogin region.
const DefaultRegion = "us"

// accessToken represents an API Access Token that will be used to authenticate
// requests.
type accessToken struct {
	token     string
	expiresAt time.Time
}

// Returns true if the token is expired.
func (t *accessToken) isExpired() bool {
	return time.Now().After(t.expiresAt)
}

// Client implements an http client around the OneLogin API
//
// https://developers.onelogin.com/api-docs/1/getting-started/dev-overview
type Client struct {
	Region string

	ClientID, ClientSecret string

	mu          sync.Mutex // Protects the access token.
	accessToken *accessToken

	client *http.Client
}

func New(c *http.Client) *Client {
	if c == nil {
		c = http.DefaultClient
	}

	return &Client{
		client: c,
	}
}

// ResponseMeta represents a OneLogin API response.
type ResponseMeta struct {
	Status struct {
		Error   bool   `json:"error"`
		Code    int    `json:"code"`
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"status"`
}

// Error is returned when the Onelogin API returns an error.
type Error struct {
	ResponseMeta
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("onelogin: unexpected response %d (%s): %s", e.Status.Code, e.Status.Type, e.Status.Message)
}

type GenerateTokenOptions struct {
	GrantType string `json:"grant_type"`
}

type GenerateTokenResponse struct {
	ResponseMeta
	Data []*GenerateTokenData `json:"data"`
}

type GenerateTokenData struct {
	AccessToken  string    `json:"access_token"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	AccountID    int       `json:"account_id"`
}

func (c *Client) GenerateToken(options GenerateTokenOptions) (*GenerateTokenResponse, error) {
	req, err := c.NewRequest("POST", "/auth/oauth2/token", options)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("client_id:%s, client_secret:%s", c.ClientID, c.ClientSecret))

	var resp GenerateTokenResponse
	_, err = c.do(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

type GenerateSAMLAssertionOptions struct {
	UsernameOrEmail string  `json:"username_or_email"`
	Password        string  `json:"password"`
	AppID           string  `json:"app_id"`
	Subdomain       string  `json:"subdomain"`
	IPAddress       *string `json:"ip_address,omitempty"`
}

type GenerateSAMLAssertionResponse struct {
	ResponseMeta
	Data interface{} `json:"data"`
}

func (r *GenerateSAMLAssertionResponse) UnmarshalJSON(b []byte) error {
	type alias GenerateSAMLAssertionResponse
	var a alias

	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}

	if a.Status.Message == "MFA is required for this user" {
		var data struct {
			Data []*GenerateSAMLAssertionMFAData `json:"data"`
		}
		if err := json.Unmarshal(b, &data); err != nil {
			return err
		}
		a.Data = data.Data
	}

	*r = GenerateSAMLAssertionResponse(a)

	return nil
}

type Device struct {
	DeviceID       int     `json:"device_id"`
	DeviceType     string  `json:"device_type"`
	DuoSigRequest  *string `json:"duo_sig_request"`
	DuoApiHostname *string `json:"duo_api_hostname"`
}

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
}

type GenerateSAMLAssertionMFAData struct {
	StateToken  string    `json:"state_token"`
	Devices     []*Device `json:"devices"`
	CallbackURL string    `json:"callback_url"`
	User        *User     `json:"user"`
}

// GenerateSAMLAssertion generates a SAML assertion and returns the response
func (c *Client) GenerateSAMLAssertion(options GenerateSAMLAssertionOptions) (*GenerateSAMLAssertionResponse, error) {
	req, err := c.NewRequest("POST", "/api/1/saml_assertion", options)
	if err != nil {
		return nil, err
	}

	var resp GenerateSAMLAssertionResponse
	_, err = c.Do(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

type VerifyFactorSAMLOptions struct {
	AppID      string  `json:"app_id"`
	DeviceID   string  `json:"device_id"`
	StateToken string  `json:"state_token"`
	OTPToken   *string `json:"otp_token,omitempty"`
}

type VerifyFactorSAMLResponse struct {
	ResponseMeta
	Data string `json:"data"`
}

func (c *Client) VerifyFactorSAML(options VerifyFactorSAMLOptions) (*VerifyFactorSAMLResponse, error) {
	req, err := c.NewRequest("POST", "/api/1/saml_assertion/verify_factor", options)
	if err != nil {
		return nil, err
	}

	var resp VerifyFactorSAMLResponse
	_, err = c.Do(req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) NewRequest(method, path string, v interface{}) (*http.Request, error) {
	region := c.Region
	if region == "" {
		region = DefaultRegion
	}

	var r io.Reader
	switch v := v.(type) {
	case io.Reader:
		r = v
	default:
		if v != nil {
			raw, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			r = bytes.NewReader(raw)
		}
	}

	req, err := http.NewRequest(method, fmt.Sprintf("https://api.%s.onelogin.com/%s", region, path), r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	if err := c.authorize(req); err != nil {
		return nil, err
	}

	return c.do(req, v)
}

func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		defer resp.Body.Close()
		var e Error
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			return resp, fmt.Errorf("onelogin: decode error: %v", err)
		}
		return resp, &e
	}

	if v != nil {
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
			return resp, fmt.Errorf("onelogin: decode response: %v", err)
		}
	}

	return resp, nil
}

func (c *Client) authorize(req *http.Request) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken == nil || c.accessToken.isExpired() {
		resp, err := c.GenerateToken(GenerateTokenOptions{
			GrantType: "client_credentials",
		})
		if err != nil {
			return fmt.Errorf("generate client credentials (client_id %q): %v", c.ClientID, err)
		}

		d := resp.Data[0]
		c.accessToken = &accessToken{
			token:     d.AccessToken,
			expiresAt: d.CreatedAt.Add(time.Duration(d.ExpiresIn) * time.Second),
		}
	}

	req.Header.Set("Authorization", fmt.Sprintf("bearer:%s", c.accessToken.token))
	return nil
}
