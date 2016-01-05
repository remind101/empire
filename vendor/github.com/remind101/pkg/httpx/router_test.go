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
				r.Handle("/path", HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					io.WriteString(w, ctx.Value("string").(string))
					return nil
				})).Methods("GET")
			},
			req:  newRequest("GET", "/path", nil),
			body: "foo",
		},

		// A headers based route.
		{
			routes: func(r *Router) {
				r.Headers("X-Foo", "bar").HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					io.WriteString(w, ctx.Value("string").(string))
					return nil
				})
			},
			req: func() *http.Request {
				r := newRequest("GET", "/path", nil)
				r.Header.Set("X-Foo", "bar")
				return r
			}(),
			body: "foo",
		},

		// A request with vars.
		{
			routes: func(r *Router) {
				r.Handle("/path/{app}", HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					vars := Vars(ctx)
					io.WriteString(w, vars["app"])
					return nil
				})).Methods("GET")
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

		// Pulling out current route.
		{
			routes: func(r *Router) {
				r.Handle("/path", HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					io.WriteString(w, RouteFromContext(ctx).GetName())
					return nil
				})).Methods("GET").Name("bar")
			},
			req:  newRequest("GET", "/path", nil),
			body: "bar",
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
