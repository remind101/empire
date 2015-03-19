package logger

import (
	"bytes"
	"log"
	"testing"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		in  []interface{}
		out string
	}{
		{[]interface{}{"key", "value"}, "key=value\n"},
		{[]interface{}{"this is a message"}, "this is a message\n"},
		{[]interface{}{"key", "value", "message"}, "key=value message\n"},
		{[]interface{}{"count", 1}, "count=1\n"},
		{[]interface{}{"b", 1, "a", 1}, "b=1 a=1\n"},
		{[]interface{}{}, "\n"},
	}

	for _, tt := range tests {
		out := testLog(tt.in...)
		if got, want := out, tt.out; got != want {
			t.Fatalf("Log => %q; want %q", got, want)
		}
	}
}

func testLog(pairs ...interface{}) string {
	b := new(bytes.Buffer)
	l := New(log.New(b, "", 0))
	l.Log(pairs...)
	return b.String()
}
