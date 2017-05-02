package honeybadger

import (
	"strings"
	"testing"
)

func TestNewErrorTrace(t *testing.T) {
	fn := func() Error {
		return NewError("Error msg")
	}

	err := fn()
	if len(err.Stack) < 3 {
		t.Errorf("Expected to generate full trace")
	}

	method := strings.Split(err.Stack[1].Method, ".")
	if method[len(method)-1] != "TestNewErrorTrace" {
		t.Errorf("Expected to generate a proper trace")
	}
}
