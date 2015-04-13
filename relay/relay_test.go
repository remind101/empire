package relay

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/remind101/empire/relay/tcptest"
)

func TestPostContainer(t *testing.T) {
	ts := NewTestHTTPServer()
	defer ts.Close()

	res, err := http.Post(ts.URL+"/containers", "application/json", strings.NewReader(`{}`))
	if err != nil {
		log.Fatal(err)
	}

	if got, want := res.StatusCode, 201; got != want {
		t.Fatalf("StatusCode => %v; want %v	", got, want)
	}
}

func NewTestHTTPServer() *httptest.Server {
	return httptest.NewServer(NewHTTPHandler(NewTestRelay()))
}

func NewTestTCPServer() *tcptest.Server {
	return tcptest.NewServer(NewTCPHandler(NewTestRelay()))
}

func NewTestRelay() *Relay {
	return New(DefaultOptions)
}
