package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
)

func TestAuthentication(t *testing.T) {
	m := &Authentication{
		finder: &mockEmpire{
			AccessTokensFindFunc: func(token string) (*empire.AccessToken, error) {
				return &empire.AccessToken{
					User: &empire.User{
						Name: "ehjolmes",
					},
				}, nil
			},
		},
		handler: HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			_, ok := empire.UserFromContext(ctx)
			if !ok {
				t.Fatal("Expected a user to be present in the context")
			}

			return nil
		}),
	}

	ctx := context.Background()
	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/apps", nil)
	req.SetBasicAuth("", "token")

	if err := m.ServeHTTPContext(ctx, resp, req); err != nil {
		t.Fatal(err)
	}
}
