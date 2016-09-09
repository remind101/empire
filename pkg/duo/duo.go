package duo

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const rfc2822 = "Mon Jan 02 15:04:05 -0700 2006"

// Error format used by Duo.
type Error struct {
	Stat          string `json:"stat"`
	Code          int    `json:"code"`
	Message       string `json:"message"`
	MessageDetail string `json:"message_detail"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("duo api error: %d: %s: %s", e.Code, e.Message, e.MessageDetail)
}

// Client is an http client for the Duo API.
type Client struct {
	Key, Secret, Host string

	client *http.Client
}

// New returns a new Client instance. If no http.Client is provided, a new
// http.Client will be initialized.
func New(c *http.Client) *Client {
	if c == nil {
		c = http.DefaultClient
	}

	return &Client{
		client: c,
	}
}

type AuthResponse struct {
	Stat     string `json:"stat"`
	Response struct {
		Result        string `json:"result"`
		Status        string `json:"status"`
		StatusMessage string `json:"status_msg"`
	}
}

// Auth performs an /auth/v2/auth action.
func (c *Client) Auth(params url.Values) (*AuthResponse, error) {
	var response AuthResponse
	_, err := c.Post("/auth/v2/auth", params, &response)
	return &response, err
}

func (c *Client) Post(path string, params url.Values, v interface{}) (*http.Response, error) {
	return c.request("POST", path, params, v)
}

func (c *Client) request(method, path string, params url.Values, v interface{}) (*http.Response, error) {
	req, err := c.NewRequest(method, path, nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = params.Encode()
	return c.Do(req, v)
}

func (c *Client) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("https://%s%s", c.Host, path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, err
}

func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	SetBasicAuth(c.Key, c.Secret, req)
	resp, err := c.client.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		var duoErr Error
		if err := json.NewDecoder(resp.Body).Decode(&duoErr); err != nil {
			return resp, err
		}
		return resp, &duoErr
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return resp, err
	}
	return resp, err
}

// SignRequest HMAC signs the http.Request, based on the alrgorithm in https://duo.com/docs/authapi
//
// The API uses HTTP Basic Authentication to authenticate requests. Use your Duo application’s integration key as the HTTP Username.
//
//	Generate the HTTP Password as an HMAC signature of the request. This will be different for each request and must be re-generated each time.
//
//	To construct the signature, first build an ASCII string from your request, using the following components:
//
//	date
//		The current time, formatted as RFC 2822. This must be the same string as the “Date” header. | Tue, 21 Aug 2012 17:29:18 -0000
//	method
//		The HTTP method (uppercase) | POST
//	host
//		Your API hostname (lowercase) | api-xxxxxxxx.duosecurity.com
//	path
//		The specific API method's path | /accounts/v1/account/list
//	params
//		The URL-encoded list of key=value pairs, lexicographically
//		sorted by key. These come from the request parameters (the URL
//		query string for GET and DELETE requests or the request body for
//		POST requests). If the request does not have any parameters one
//		must still include a blank line in the string that is signed. Do
//		not encode unreserved characters. Use upper-case hexadecimal
//		digits A through F in escape sequences.
func Signature(secret string, r *http.Request) string {
	if r.Header.Get("Date") == "" {
		r.Header.Set("Date", time.Now().Format(rfc2822))
	}

	body := signatureBody(r)
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write(body)
	return fmt.Sprintf("%x", mac.Sum(nil))
}

// SetBasicAuth signs the request, then sets the Authorization header.
func SetBasicAuth(key, secret string, r *http.Request) {
	signature := Signature(secret, r)
	r.SetBasicAuth(key, signature)
}

func signatureBody(r *http.Request) []byte {
	parts := []string{
		r.Header.Get("Date"),
		strings.ToUpper(r.Method),
		strings.ToLower(r.URL.Host),
		r.URL.Path,
		r.URL.RawQuery,
	}
	return []byte(strings.Join(parts, "\n"))
}
