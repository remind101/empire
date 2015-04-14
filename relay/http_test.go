package relay

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestPostContainer(t *testing.T) {
	ts := NewTestHTTPServer(nil)
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
