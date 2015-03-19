package logger

import (
	"bytes"
	"testing"
)

func TestLogger(t *testing.T) {
	tests := []struct {
		in  []interface{}
		out string
	}{
		{[]interface{}{"key", "value"}, "request_id=1234 key=value\n"},
		{[]interface{}{"this is a message"}, "request_id=1234 this is a message\n"},
		{[]interface{}{"key", "value", "message"}, "request_id=1234 key=value message\n"},
		{[]interface{}{"count", 1}, "request_id=1234 count=1\n"},
		{[]interface{}{"b", 1, "a", 1}, "request_id=1234 b=1 a=1\n"},
		{[]interface{}{}, "request_id=1234\n"},
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
	l := New(b, "1234")
	l.Log(pairs...)
	return b.String()
}
