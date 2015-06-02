package reporter

import (
	"errors"
	"net/http"
	"path"
	"runtime"
	"testing"

	"golang.org/x/net/context"
)

var errBoom = errors.New("boom")

func TestReport(t *testing.T) {
	r := ReporterFunc(func(ctx context.Context, err error) error {
		e := err.(*Error)

		if e.Request.Header.Get("Content-Type") != "application/json" {
			t.Fatal("request information not set")
		}

		checkFirstFunc(t, e, "reporter.TestReport")

		return nil
	})
	ctx := WithReporter(context.Background(), r)

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Content-Type", "application/json")
	AddRequest(ctx, req)

	if err := Report(ctx, errBoom); err != nil {
		t.Fatal(err)
	}
}

func TestReportWithSkip(t *testing.T) {
	r := ReporterFunc(func(ctx context.Context, err error) error {
		e := err.(*Error)

		checkFirstFunc(t, e, "reporter.TestReportWithSkip")

		return nil
	})
	ctx := WithReporter(context.Background(), r)

	func() {
		if err := ReportWithSkip(ctx, errBoom, 1); err != nil {
			t.Fatal(err)
		}
	}()
}

func checkFirstFunc(t testing.TB, err *Error, name string) {
	line := err.Backtrace[0]
	fn := runtime.FuncForPC(line.PC)

	if got, want := path.Base(fn.Name()), name; got != want {
		t.Fatalf("Function => %s; want %s", got, want)
	}
}
