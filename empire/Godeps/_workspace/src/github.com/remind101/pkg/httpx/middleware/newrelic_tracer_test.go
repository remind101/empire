package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remind101/newrelic"
	"github.com/remind101/pkg/httpx"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

type tracerTest struct {
	// A function that adds Handlers to the router.
	routes func(*httpx.Router)

	// An http.Request to test.
	req *http.Request

	expectedTransactionName string
	expectedUrl             string
}

func TestTracing(t *testing.T) {
	tracerTests := []tracerTest{
		// simple path
		{
			routes: func(r *httpx.Router) {
				r.Handle("/path", httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					return nil
				})).Methods("GET")
			},
			req: newRequest("GET", "/path"),
			expectedTransactionName: "GET /path",
			expectedUrl:             "/path",
		},
		// path with variables
		{
			routes: func(r *httpx.Router) {
				r.Handle("/users/{user_id}", httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					return nil
				})).Methods("DELETE")
			},
			req: newRequest("DELETE", "/users/23"),
			expectedTransactionName: "DELETE /users/{user_id}",
			expectedUrl:             "/users/23",
		},
		// path with regexp variables
		{
			routes: func(r *httpx.Router) {
				r.Handle("/articles/{category}/{id:[0-9]+}", httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					return nil
				})).Methods("PUT")
			},
			req: newRequest("PUT", "/articles/tech/123"),
			expectedTransactionName: "PUT /articles/{category}/{id:[0-9]+}",
			expectedUrl:             "/articles/tech/123",
		},
		// using Path().Handler() style
		{
			routes: func(r *httpx.Router) {
				r.Path("/articles/{category}/{id}").HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
					return nil
				}).Methods("GET")
			},
			req: newRequest("GET", "/articles/tech/456"),
			expectedTransactionName: "GET /articles/{category}/{id}",
			expectedUrl:             "/articles/tech/456",
		},
		// no route
		{
			routes: func(r *httpx.Router) {
			},
			req: newRequest("GET", "/non_existent"),
			expectedTransactionName: "GET /non_existent",
			expectedUrl:             "/non_existent",
		},
	}

	for _, tt := range tracerTests {
		traceTest(t, &tt)
	}
}

func traceTest(t *testing.T, tt *tracerTest) {
	var m httpx.Handler
	r := httpx.NewRouter()

	if tt.routes != nil {
		tt.routes(r)
	}

	tx := new(mockTx)
	m = &NewRelicTracer{
		handler: r,
		router:  r,
		tracer:  nil,
		createTx: func(transactionName, url string, tracer newrelic.TxTracer) newrelic.Tx {
			if tt.expectedTransactionName != transactionName {
				t.Fatalf("Transaction mismatch expected: %v got: %v", tt.expectedTransactionName, transactionName)
			}
			if tt.expectedUrl != url {
				t.Fatalf("Url mismatch expected: %v got: %v", tt.expectedUrl, url)
			}
			return tx
		},
	}

	ctx := context.Background()
	resp := httptest.NewRecorder()

	tx.On("Start").Return(nil)
	tx.On("End").Return(nil)

	if err := m.ServeHTTPContext(ctx, resp, tt.req); err != nil {
		t.Fatal(err)
	}

	tx.AssertExpectations(t)
}

func newRequest(method, path string) *http.Request {
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}

	return req
}

type mockTx struct {
	mock.Mock
}

func (t *mockTx) Start() error {
	args := t.Called()
	return args.Error(0)
}

func (t *mockTx) End() error {
	args := t.Called()
	return args.Error(0)
}

func (t *mockTx) StartGeneric(name string) error {
	t.Called(name)
	return nil
}

func (t *mockTx) StartDatastore(table, operation, sql, rollupName string) error {
	t.Called(table, operation, sql, rollupName)
	return nil
}

func (t *mockTx) StartExternal(host, name string) error {
	t.Called(host, name)
	return nil
}

func (t *mockTx) EndSegment() error {
	args := t.Called()
	return args.Error(0)
}

func (t *mockTx) ReportError(exceptionType, errorMessage, stackTrace, stackDelim string) error {
	args := t.Called()
	return args.Error(0)
}
