package empire

import "testing"

func TestRepoDomain(t *testing.T) {
	tests := []struct {
		in  Repo
		out string
	}{
		{"ejholmes/acme-inc", ""},
		{"quay.io/ejholmes/acme-inc", "quay.io"},
	}

	for _, tt := range tests {
		out := tt.in.Domain()

		if got, want := out, tt.out; got != want {
			t.Fatalf("%q.Domain() => %s; want %s", tt.in, got, want)
		}
	}
}

func TestRepoPath(t *testing.T) {
	tests := []struct {
		in  Repo
		out string
	}{
		{"ejholmes/acme-inc", "ejholmes/acme-inc"},
		{"quay.io/ejholmes/acme-inc", "ejholmes/acme-inc"},
	}

	for _, tt := range tests {
		out := tt.in.Path()

		if got, want := out, tt.out; got != want {
			t.Fatalf("%q.Path() => %s; want %s", tt.in, got, want)
		}
	}
}
