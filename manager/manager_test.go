package manager

import "testing"

func TestNewName(t *testing.T) {
	n := NewName("1234", "v1", "web", 1)

	if got, want := n, Name("1234.v1.web.1"); got != want {
		t.Fatalf("Name => %v; want %v", got, want)
	}
}
