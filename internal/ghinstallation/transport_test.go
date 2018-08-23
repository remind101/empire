package ghinstallation

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

const (
	installationID = 1
	integrationID  = 2
	token          = "abc123"
)

var key = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEA0BUezcR7uycgZsfVLlAf4jXP7uFpVh4geSTY39RvYrAll0yh
q7uiQypP2hjQJ1eQXZvkAZx0v9lBYJmX7e0HiJckBr8+/O2kARL+GTCJDJZECpjy
97yylbzGBNl3s76fZ4CJ+4f11fCh7GJ3BJkMf9NFhe8g1TYS0BtSd/sauUQEuG/A
3fOJxKTNmICZr76xavOQ8agA4yW9V5hKcrbHzkfecg/sQsPMmrXixPNxMsqyOMmg
jdJ1aKr7ckEhd48ft4bPMO4DtVL/XFdK2wJZZ0gXJxWiT1Ny41LVql97Odm+OQyx
tcayMkGtMb1nwTcVVl+RG2U5E1lzOYpcQpyYFQIDAQABAoIBAAfUY55WgFlgdYWo
i0r81NZMNBDHBpGo/IvSaR6y/aX2/tMcnRC7NLXWR77rJBn234XGMeQloPb/E8iw
vtjDDH+FQGPImnQl9P/dWRZVjzKcDN9hNfNAdG/R9JmGHUz0JUddvNNsIEH2lgEx
C01u/Ntqdbk+cDvVlwuhm47MMgs6hJmZtS1KDPgYJu4IaB9oaZFN+pUyy8a1w0j9
RAhHpZrsulT5ThgCra4kKGDNnk2yfI91N9lkP5cnhgUmdZESDgrAJURLS8PgInM4
YPV9L68tJCO4g6k+hFiui4h/4cNXYkXnaZSBUoz28ICA6e7I3eJ6Y1ko4ou+Xf0V
csM8VFkCgYEA7y21JfECCfEsTHwwDg0fq2nld4o6FkIWAVQoIh6I6o6tYREmuZ/1
s81FPz/lvQpAvQUXGZlOPB9eW6bZZFytcuKYVNE/EVkuGQtpRXRT630CQiqvUYDZ
4FpqdBQUISt8KWpIofndrPSx6JzI80NSygShQsScWFw2wBIQAnV3TpsCgYEA3reL
L7AwlxCacsPvkazyYwyFfponblBX/OvrYUPPaEwGvSZmE5A/E4bdYTAixDdn4XvE
ChwpmRAWT/9C6jVJ/o1IK25dwnwg68gFDHlaOE+B5/9yNuDvVmg34PWngmpucFb/
6R/kIrF38lEfY0pRb05koW93uj1fj7Uiv+GWRw8CgYEAn1d3IIDQl+kJVydBKItL
tvoEur/m9N8wI9B6MEjhdEp7bXhssSvFF/VAFeQu3OMQwBy9B/vfaCSJy0t79uXb
U/dr/s2sU5VzJZI5nuDh67fLomMni4fpHxN9ajnaM0LyI/E/1FFPgqM+Rzb0lUQb
yqSM/ptXgXJls04VRl4VjtMCgYEAprO/bLx2QjxdPpXGFcXbz6OpsC92YC2nDlsP
3cfB0RFG4gGB2hbX/6eswHglLbVC/hWDkQWvZTATY2FvFps4fV4GrOt5Jn9+rL0U
elfC3e81Dw+2z7jhrE1ptepprUY4z8Fu33HNcuJfI3LxCYKxHZ0R2Xvzo+UYSBqO
ng0eTKUCgYEAxW9G4FjXQH0bjajntjoVQGLRVGWnteoOaQr/cy6oVii954yNMKSP
rezRkSNbJ8cqt9XQS+NNJ6Xwzl3EbuAt6r8f8VO1TIdRgFOgiUXRVNZ3ZyW8Hegd
kGTL0A6/0yAu9qQZlFbaD5bWhQo7eyx63u4hZGppBhkTSPikOYUPCH8=
-----END RSA PRIVATE KEY-----`)

func TestNew(t *testing.T) {

	var authed bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != acceptHeader {
			t.Fatalf("Request URI %q accept header got %q want: %q", r.RequestURI, r.Header.Get("Accept"), acceptHeader)
		}
		switch r.RequestURI {
		case fmt.Sprintf("/installations/%d/access_tokens", installationID):
			// respond with any token to installation transport
			js, _ := json.Marshal(accessToken{
				Token:     token,
				ExpiresAt: time.Now().Add(5 * time.Minute),
			})
			fmt.Fprintln(w, string(js))
			authed = true
		case "/auth/with/installation/token/endpoint":
			if want := "token " + token; r.Header.Get("Authorization") != want {
				t.Fatalf("Installation token got: %q want: %q", r.Header.Get("Authorization"), want)
			}
		default:
			t.Fatalf("unexpected URI: %q", r.RequestURI)
		}
	}))
	defer ts.Close()

	tr, err := New(&http.Transport{}, integrationID, installationID, key)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	tr.BaseURL = ts.URL

	client := http.Client{Transport: tr}
	_, err = client.Get(ts.URL + "/auth/with/installation/token/endpoint")
	if err != nil {
		t.Fatal("unexpected error from client:", err)
	}

	if !authed {
		t.Fatal("Expected fetch of access_token but none occurred")
	}

	// Check the token is reused by setting expires into far future
	tr.token.ExpiresAt = time.Now().Add(time.Hour)
	authed = false

	_, err = client.Get(ts.URL + "/auth/with/installation/token/endpoint")
	if err != nil {
		t.Fatal("unexpected error from client:", err)
	}

	if authed {
		t.Fatal("Unexpected fetch of access_token")
	}

	// Check the token is refreshed by setting expires into far past
	tr.token.ExpiresAt = time.Unix(0, 0)

	_, err = client.Get(ts.URL + "/auth/with/installation/token/endpoint")
	if err != nil {
		t.Fatal("unexpected error from client:", err)
	}

	if !authed {
		t.Fatal("Expected fetch of access_token but none occurred")
	}
}

func TestNewKeyFromFile(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(key); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = NewKeyFromFile(&http.Transport{}, integrationID, installationID, tmpfile.Name())
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
}

func TestNew_appendHeader(t *testing.T) {
	var headers http.Header
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header
		fmt.Fprintln(w, `{}`) // dummy response that looks like json
	}))
	defer ts.Close()

	// Create a new request adding our own Accept header
	myheader := "my-header"
	req, err := http.NewRequest("GET", ts.URL+"/auth/with/installation/token/endpoint", nil)
	if err != nil {
		t.Fatal("unexpected error from http.NewRequest:", err)
	}
	req.Header.Add("Accept", myheader)

	tr, err := New(&http.Transport{}, integrationID, installationID, key)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	tr.BaseURL = ts.URL

	client := http.Client{Transport: tr}
	_, err = client.Do(req)
	if err != nil {
		t.Fatal("unexpected error from client:", err)
	}

	found := false
	for _, v := range headers["Accept"] {
		if v == myheader {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("could not find %v in request's accept headers: %v", myheader, headers["Accept"])
	}
}
