package duo

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testIntegrationKey    = "DIWJ8X6AEYOR5OMC6TQ1"
	testIntegrationSecret = "Zh5eGmUq9zpfQnyUIu5OL9iWoMMv5ZNmk3zLJ4Ep"
)

func TestSignature(t *testing.T) {
	tests := []struct {
		req       *http.Request
		signature string
	}{
		{
			newRequest("POST", "https://api-XXXXXXXX.duosecurity.com/accounts/v1/account/list", func(r *http.Request) {
				r.URL.RawQuery = "realname=First%20Last&username=root"
				r.Header.Set("Date", "Tue, 21 Aug 2012 17:29:18 -0000")
			}),
			"2d97d6166319781b5a3a07af39d366f491234edc",
		},
		{
			newRequest("POST", "https://api-XXXXXXXX.duosecurity.com/accounts/v1/account/list", func(r *http.Request) {
				q := url.Values{}
				q.Add("realname", "First Last")
				q.Add("username", "root")
				r.URL.RawQuery = q.Encode()
				r.Header.Set("Date", "Tue, 21 Aug 2012 17:29:18 -0000")
			}),
			"c423a17fe94ccbf3004533cab496ae06216b90be",
		},
		{
			newRequest("POST", "https://api-XXXXXXXX.duosecurity.com/accounts/v1/account/list", func(r *http.Request) {
				r.Header.Set("Date", "Tue, 21 Aug 2012 17:29:18 -0000")
			}),
			"1f3107d3856797e459d29a440c6e79ead6b7163f",
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			signature := Signature(testIntegrationSecret, tt.req)
			assert.Equal(t, tt.signature, signature)
		})
	}
}

func TestSetBasicAuth(t *testing.T) {
	tests := []struct {
		req           *http.Request
		authorization string
	}{
		{
			newRequest("POST", "https://api-XXXXXXXX.duosecurity.com/accounts/v1/account/list", func(r *http.Request) {
				r.URL.RawQuery = "realname=First%20Last&username=root"
				r.Header.Set("Date", "Tue, 21 Aug 2012 17:29:18 -0000")
			}),
			"Basic RElXSjhYNkFFWU9SNU9NQzZUUTE6MmQ5N2Q2MTY2MzE5NzgxYjVhM2EwN2FmMzlkMzY2ZjQ5MTIzNGVkYw==",
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			SetBasicAuth(testIntegrationKey, testIntegrationSecret, tt.req)
			assert.Equal(t, tt.authorization, tt.req.Header.Get("Authorization"))
		})
	}
}

func newRequest(method, path string, f func(r *http.Request)) *http.Request {
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}
	if f != nil {
		f(req)
	}
	return req
}
