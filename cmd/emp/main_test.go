package main

import (
	"net/http"
	"os"
	"testing"
)

func init() {
	os.Setenv("EMPIRE_API_URL", "http://localhost:8080")
}

func TestSSLEnabled(t *testing.T) {
	initClients()

	if client.HTTP == nil {
		// No http.Client means the client defaults to SSL enabled
		return
	}
	if client.HTTP.Transport == nil {
		// No transport means the client defaults to SSL enabled
		return
	}
	conf := client.HTTP.Transport.(*http.Transport).TLSClientConfig
	if conf == nil {
		// No TLSClientConfig means the client defaults to SSL enabled
		return
	}
	if conf.InsecureSkipVerify {
		t.Errorf("expected InsecureSkipVerify == false")
	}

	client = nil
}

func TestSSLDisable(t *testing.T) {
	os.Setenv("HEROKU_SSL_VERIFY", "disable")
	initClients()

	if client.HTTP == nil {
		t.Fatalf("client.HTTP not set, expected http.Client")
	}
	if client.HTTP.Transport == nil {
		t.Fatalf("client.HTTP.Transport not set")
	}
	conf := client.HTTP.Transport.(*http.Transport).TLSClientConfig
	if conf == nil {
		t.Fatalf("client.HTTP.Transport's TLSClientConfig is nil")
	}
	if !conf.InsecureSkipVerify {
		t.Errorf("expected InsecureSkipVerify == true")
	}

	os.Setenv("HEROKU_SSL_VERIFY", "")
	client = nil
}

func TestHerokuAPIURL(t *testing.T) {
	newURL := "https://api.otherheroku.com"
	os.Setenv("EMPIRE_API_URL", newURL)
	initClients()

	if client.URL != newURL {
		t.Errorf("expected client.URL to be %q, got %q", newURL, client.URL)
	}

	if apiURL != newURL {
		t.Errorf("expected apiURL to be %q, got %q", newURL, apiURL)
	}

	// cleanup
	os.Setenv("EMPIRE_API_URL", "")
}
