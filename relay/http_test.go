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
	ex := "{\"image\":\"phusion/baseimage\",\"name\":\"run.1\",\"command\":\"/bin/bash\",\"state\":\"starting\",\"env\":{\"TERM\":\"x-term\"},\"attach\":true,\"attach_url\":\"rendezvous://rendez.empire.example.com:5000/1\"}\n"
	if got, want := bs, ex; got != want {
		t.Fatalf("res.Body => %q; want %q", got, want)
	}
}
