package trace

import (
	"testing"
	"time"

	"golang.org/x/net/context"
)

func TestTracedContext(t *testing.T) {
	ctx := &tracedContext{Context: context.Background(), id: "1234", file: "pkg/main.go", line: 10, fnname: "pkg/main.main", parent: "4321"}

	if id, ok := ctx.Value("trace.id").(string); !ok || id != "1234" {
		t.Fatalf("Expected id; got %v", id)
	}

	if id, ok := ctx.Value("trace.parent").(string); !ok || id != "4321" {
		t.Fatalf("Expected parent; got %v", id)
	}

	if fnname, ok := ctx.Value("trace.func").(string); !ok || fnname != "pkg/main.main" {
		t.Fatalf("Expected func name; got %v", fnname)
	}

	if file, ok := ctx.Value("trace.file").(string); !ok || file != "pkg/main.go" {
		t.Fatalf("Expected file; got %v", file)
	}

	if duration, ok := ctx.Value("trace.duration").(time.Duration); !ok {
		t.Fatalf("Expected a duration; got %v", duration)
	}
}
