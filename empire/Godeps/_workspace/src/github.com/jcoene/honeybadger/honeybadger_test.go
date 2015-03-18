package honeybadger

import (
	"testing"
)

func TestExceptionClass(t *testing.T) {
	var result string
	result = exceptionClass("")
	if result != "" {
		t.Fatalf("Should have been '' but was '%s'", result)
	}

	result = exceptionClass("test")
	if result != "test" {
		t.Fatalf("Should have been 'test' but was '%s'", result)
	}

	result = exceptionClass("honeybadger: failed because: io error")
	if result != "io error" {
		t.Fatalf("Should have been 'io error' but was '%s'", result)
	}

	result = exceptionClass("honeybadger: failed because:")
	if result != "failed because" {
		t.Fatalf("Should have been 'failed because' but was '%s'", result)
	}

	// test space trimming
	result = exceptionClass("honeybadger: failed because :    ")
	if result != "failed because" {
		t.Fatalf("Should have been 'failed because' but was '%s'", result)
	}

	// test space trimming
	result = exceptionClass("honeybadger::")
	if result != "honeybadger" {
		t.Fatalf("Should have been 'honeybadger' but was '%s'", result)
	}

}

func TestFullMessage(t *testing.T) {
	var result string
	result = fullMessage("foo bar")
	if result != "foo bar" {
		t.Fatalf("Should have been 'foo bar' but was '%s'", result)
	}
}
