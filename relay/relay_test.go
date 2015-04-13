package relay

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/remind101/empire/relay/tcptest"
)

func TestPostContainer(t *testing.T) {
	ts := NewTestHTTPServer()
	defer ts.Close()

	body := `{"image":"phusion/baseimage", "command":"/bin/bash", "env": { "TERM":"x-term"}, "attach":true}`

	res, err := http.Post(ts.URL+"/containers", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	if got, want := res.StatusCode, 201; got != want {
		t.Fatalf("res.StatusCode => %v; want %v	", got, want)
	}

	bb, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	bs := string(bb)
	if got, want := bs, "{\"attach_url\":\"rendezvous://rendez.empire.example.com/1\"}\n"; got != want {
		t.Fatalf("res.Body => %q; want %q", got, want)
	}
}

func NewTestHTTPServer() *httptest.Server {
	return httptest.NewServer(NewHTTPHandler(NewTestRelay()))
}

func NewTestTCPServer() *tcptest.Server {
	return tcptest.NewServer(NewTCPHandler(NewTestRelay()))
}

func NewTestRelay() *Relay {
	o := DefaultOptions
	o.Host = "rendezvous://rendez.empire.example.com"
	sid := 0
	o.SessionGenerator = func() string {
		sid++
		return fmt.Sprintf("%d", sid)
	}
	return New(o)
}
