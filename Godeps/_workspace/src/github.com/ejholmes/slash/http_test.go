package slash

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"github.com/stretchr/testify/assert"
)

func TestServer_Reply(t *testing.T) {
	h := HandlerFunc(func(ctx context.Context, r Responder, command Command) error {
		return nil
	})
	s := &Server{
		Handler: h,
	}

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(testForm))

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "", resp.Body.String())
}

func TestServer_Say(t *testing.T) {
	h := HandlerFunc(func(ctx context.Context, r Responder, command Command) error {
		return nil
	})
	s := &Server{
		Handler: h,
	}

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(testForm))

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "", resp.Body.String())
}

func TestServer_Err(t *testing.T) {
	h := HandlerFunc(func(ctx context.Context, r Responder, command Command) error {
		return errors.New("boom")
	})
	s := &Server{
		Handler: h,
	}

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", strings.NewReader(testForm))

	s.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
}
