package hb

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"
)

var errBoom = errors.New("boom")

func TestSend(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer s.Close()

	r := &Reporter{
		client: &Client{
			URL: s.URL,
		},
	}

	if err := r.Report(context.Background(), errBoom); err != nil {
		t.Fatal(err)
	}
}

func TestReportGenerator(t *testing.T) {
	g := NewReportGenerator("test")

	report, err := g.Generate(context.Background(), errBoom)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(raw))
}
