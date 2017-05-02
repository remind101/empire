package honeybadger

import "testing"

func TestNewConfig(t *testing.T) {
	client := New(Configuration{APIKey: "lemmings"})
	if client.Config.APIKey != "lemmings" {
		t.Errorf("Expected New to configure APIKey. expected=%#v actual=%#v", "lemmings", client.Config.APIKey)
	}
}

func TestConfigureClient(t *testing.T) {
	client := New(Configuration{})
	client.Configure(Configuration{APIKey: "badgers"})
	if client.Config.APIKey != "badgers" {
		t.Errorf("Expected Configure to override config.APIKey. expected=%#v actual=%#v", "badgers", client.Config.APIKey)
	}
}

func TestConfigureClientEndpoint(t *testing.T) {
	client := New(Configuration{})
	backend := client.Config.Backend.(*server)
	client.Configure(Configuration{Endpoint: "http://localhost:3000"})
	if *backend.URL != "http://localhost:3000" {
		t.Errorf("Expected Configure to update backend. expected=%#v actual=%#v", "http://localhost:3000", backend.URL)
	}
}

func TestClientContext(t *testing.T) {
	client := New(Configuration{})
	client.context = &Context{"foo": "bar"}

	client.SetContext(Context{"bar": "baz"})
	context := *client.context

	if context["foo"] != "bar" {
		t.Errorf("Expected client to merge global context. expected=%#v actual=%#v", "bar", context["foo"])
	}

	if context["bar"] != "baz" {
		t.Errorf("Expected client to merge global context. expected=%#v actual=%#v", "baz", context["bar"])
	}
}
