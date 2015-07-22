package hooker

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ejholmes/hookshot"
)

func TestHooker(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := string(raw), `{"Data":"foo"}`+"\n"; got != want {
			t.Fatalf("Body => %s; want %s", got, want)
		}

		io.WriteString(w, "ok\n")
	})
	s := httptest.NewServer(hookshot.Authorize(h, "secret"))
	defer s.Close()

	c := NewClient(nil)
	c.Secret = "secret"
	c.URL = s.URL

	body := struct {
		Data string
	}{
		Data: "foo",
	}
	resp, err := c.Trigger("ping", &body)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(raw), "ok\n"; got != want {
		t.Fatalf("Response => %s; want %s", got, want)
	}
}
