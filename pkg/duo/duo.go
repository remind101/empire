package duo

import (
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const rfc2822 = "Mon Jan 02 15:04:05 -0700 2006"

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
func SignRequest(secret []byte, r *http.Request) string {
	if r.Header.Get("Date") == "" {
		r.Header.Set("Date", time.Now().Format(rfc2822))
	}

	body := signatureBody(r)
	mac := hmac.New(sha1.New, secret)
	mac.Write(body)
	return fmt.Sprintf("%x", mac.Sum(nil))
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
