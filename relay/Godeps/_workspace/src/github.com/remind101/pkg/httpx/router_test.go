package httpx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"
)

type routerTest struct {
	// A function that adds Handlers to the router.
	routes func(*Router)

	// An http.Request to test.
	req *http.Request

	// The expected string body.
	body string
}

func TestRouter(t *testing.T) {
	tests := []routerTest{
		// A simple request.
		{
			routes: func(r *Router) {
				r.Handle("GET", "/path", HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					io.WriteString(w, ctx.Value("string").(string))
					return nil
				}))
			},
			req:  newRequest("GET", "/path", nil),
			body: "foo",
		},

		// A request with vars.
		{
			routes: func(r *Router) {
				r.Handle("GET", "/path/{app}", HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					vars := Vars(ctx)
					io.WriteString(w, vars["app"])
					return nil
				}))
			},
			req:  newRequest("GET", "/path/acme-inc", nil),
			body: "acme-inc",
		},

		// A not found request with no NotFoundHandler.
		{
			req:  newRequest("GET", "/", nil),
			body: "404 page not found\n",
		},

		// A not found request with a custom NotFoundHandler.
		{
			routes: func(r *Router) {
				r.NotFoundHandler = HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					io.WriteString(w, "not found")
					return nil
				})
			},
			req:  newRequest("GET", "/", nil),
			body: "not found",
		},
	}

	for i, tt := range tests {
		testRouterTest(t, &tt, i)
	}
}

func testRouterTest(t *testing.T, tt *routerTest, i int) {
	r := NewRouter()

	if tt.routes != nil {
		tt.routes(r)
	}

	ctx := context.WithValue(context.Background(), "string", "foo")
	resp := httptest.NewRecorder()

	if err := r.ServeHTTPContext(ctx, resp, tt.req); err != nil {
		t.Fatal(err)
	}

	if got, want := resp.Body.String(), tt.body; got != want {
		t.Fatalf("#%d: Body => %s; want %s", i, got, want)
	}
}

func newRequest(method, path string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		panic(err)
	}

	return req
}
