package pusher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func setupTestServer(handler http.Handler) (server *httptest.Server) {
	server = httptest.NewServer(handler)
	return
}

func verifyRequest(t *testing.T, prefix string, req *http.Request, method, path string) (payload Payload) {
	if method != req.Method {
		t.Errorf("%s: Expected method %s, got %s", prefix, method, req.Method)
	}
	if path != req.URL.Path {
		t.Errorf("%s: Expected path '%s', got '%s'", prefix, path, req.URL.Path)
	}

	err := json.NewDecoder(req.Body).Decode(&payload)
	if err != nil {
		fmt.Println("Got error:", err)
	}

	return
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestPublish(t *testing.T) {
	server := setupTestServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintf(w, "{}")

		payload := verifyRequest(t, "Publish()", request, "POST", "/apps/1/events")

		if payload.Name != "event" {
			t.Errorf("Publish(): Expected body[name] = \"event\", got %q", payload.Name)
		}
		if !reflect.DeepEqual(payload.Channels, []string{"mychannel", "c2"}) {
			t.Errorf("Publish(): Expected body[channels] = [mychannel c2], got %+v", payload.Channels)
		}
	}))
	defer server.Close()

	url, _ := url.Parse(server.URL)

	client := NewClient("1", "key", "secret")
	client.Host = url.Host
	err := client.Publish("data", "event", "mychannel", "c2")

	if err != nil {
		t.Errorf("Publish(): %v", err)
	}
}

func TestFields(t *testing.T) {
	client := NewClient("1", "key", "secret")

	if client.appid != "1" {
		t.Errorf("appid not set correctly")
	}

	if client.key != "key" {
		t.Errorf("key not set correctly")
	}

	if client.secret != "secret" {
		t.Errorf("secret not set correctly")
	}
}

func TestDefaultHost(t *testing.T) {
	client := NewClient("1", "key", "secret")

	if client.Host != "api.pusherapp.com" {
		t.Errorf("Host not set correctly")
	}
}

func TestDefaultScheme(t *testing.T) {
	client := NewClient("1", "key", "secret")

	if client.Scheme != "http" {
		t.Errorf("Scheme not set correctly")
	}
}
