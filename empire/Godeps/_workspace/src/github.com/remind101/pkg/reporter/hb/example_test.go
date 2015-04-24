package hb_test

import (
	"errors"
	"net/http"

	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/hb"
	"golang.org/x/net/context"
)

var errBoom = errors.New("boom")

func Example() {
	ctx := reporter.WithReporter(context.Background(), hb.NewReporter("dcb8affa"))
	req, _ := http.NewRequest("GET", "/api/foo", nil)
	req.Header.Set("Content-Type", "application/json")

	reporter.AddContext(ctx, "request_id", "1234")
	reporter.AddRequest(ctx, req)
	reporter.Report(ctx, errBoom)
	// Output:
}
