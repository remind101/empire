package reporter

import (
	"errors"
	"fmt"
	"net/http"
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

		stack := e.StackTrace()
		var method string
		if stack != nil && len(stack) > 0 {
			method = fmt.Sprintf("%n", stack[0])
		}

		if got, want := method, "TestReport"; got != want {
			t.Fatalf("expected the first stacktrace method to be %v, got %v", want, got)
		}

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

func TestReportWithNoReporterInContext(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Expected panic due to context without reporter, got no panic")
		}
	}()
	ctx := context.Background() // no reporter
	Report(ctx, errBoom)
}

func TestMonitor(t *testing.T) {
	ensureRepanicked := func() {
		if v := recover(); v == nil {
			t.Errorf("Must have panicked after reporting!")
		}
	}
	var reportedError error
	r := ReporterFunc(func(ctx context.Context, err error) error {
		reportedError = err
		return nil
	})
	ctx := WithReporter(context.Background(), r)

	done := make(chan interface{})
	go func() {
		defer close(done)
		defer ensureRepanicked()
		defer Monitor(ctx)
		panic("oh noes!")
	}()
	<-done
	if reportedError == nil || reportedError.Error() != "panic: oh noes!" {
		t.Errorf("expected panic 'oh noes!' to be reported, got %#v", reportedError)
	}
}
