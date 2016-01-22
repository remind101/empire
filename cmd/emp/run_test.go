package main

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDialParams(t *testing.T) {
	tests := []struct {
		url string

		proto, address string
	}{
		{"http://localhost", "tcp", "localhost:80"},
		{"http://localhost:8080", "tcp", "localhost:8080"},

		{"https://empire", "tls", "empire:443"},
		{"https://empire:8443", "tls", "empire:8443"},
	}

	for _, tt := range tests {
		url, err := url.Parse(tt.url)
		assert.NoError(t, err)

		proto, address := dialParams(url)
		assert.Equal(t, tt.proto, proto)
		assert.Equal(t, tt.address, address)
	}
}
