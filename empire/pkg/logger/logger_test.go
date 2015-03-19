package logger

import (
	"bytes"
	"testing"
)

func TestLogger(t *testing.T) {
	b := new(bytes.Buffer)
	l := New(b, "1234")

	l.Log(map[string]interface{}{
		"key": "value",
	})

	if got, want := b.String(), "request_id=1234 key=value\n"; got != want {
		t.Fatalf("Println => %s; want %s", got, want)
	}
}
