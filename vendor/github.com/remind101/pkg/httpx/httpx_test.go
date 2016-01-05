package httpx

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/context"
)

func TestRequestContext(t *testing.T) {
	req, _ := http.NewRequest("POST", "/path", strings.NewReader("body"))
	req.RequestURI = "/path"
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Request-ID", "1234")

	ctx := &requestContext{Context: context.Background(), r: req}

	if method, ok := ctx.Value("http.request.method").(string); !ok || method != "POST" {
		t.Fatalf("Expected POST method; got %v", method)
	}

	if id, ok := ctx.Value("http.request.id").(string); !ok || id != "1234" {
		t.Fatalf("Expected 1234 request id; got %v", id)
	}

	if uri, ok := ctx.Value("http.request.uri").(string); !ok || uri != "/path" {
		t.Fatalf("Expected /path; got %v", uri)
	}

	if r, ok := RequestFromContext(ctx); !ok {
		t.Fatal("Expected request to be in context")
	} else if r != req {
		t.Fatal("Expected request to match")
	}
}
