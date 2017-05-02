package rollbar

import (
	"testing"
)

func TestBuildStack(t *testing.T) {
	frame := BuildStack(1)[0]
	if frame.Filename != "github.com/stvp/rollbar/stack_test.go" {
		t.Errorf("got: %s", frame.Filename)
	}
	if frame.Method != "rollbar.TestBuildStack" {
		t.Errorf("got: %s", frame.Method)
	}
	if frame.Line != 8 {
		t.Errorf("got: %d", frame.Line)
	}
}

func TestStackFingerprint(t *testing.T) {
	tests := []struct {
		Fingerprint string
		Stack       Stack
	}{
		{
			"9344290d",
			Stack{
				Frame{"foo.go", "Oops", 1},
			},
		},
		{
			"a4d78b7",
			Stack{
				Frame{"foo.go", "Oops", 2},
			},
		},
		{
			"50e0fcb3",
			Stack{
				Frame{"foo.go", "Oops", 1},
				Frame{"foo.go", "Oops", 2},
			},
		},
	}

	for i, test := range tests {
		fingerprint := test.Stack.Fingerprint()
		if fingerprint != test.Fingerprint {
			t.Errorf("tests[%d]: got %s", i, fingerprint)
		}
	}
}

func TestShortenFilePath(t *testing.T) {
	tests := []struct {
		Given    string
		Expected string
	}{
		{"", ""},
		{"foo.go", "foo.go"},
		{"/usr/local/go/src/pkg/runtime/proc.c", "pkg/runtime/proc.c"},
		{"/home/foo/go/src/github.com/stvp/rollbar.go", "github.com/stvp/rollbar.go"},
	}
	for i, test := range tests {
		got := shortenFilePath(test.Given)
		if got != test.Expected {
			t.Errorf("tests[%d]: got %s", i, got)
		}
	}
}
