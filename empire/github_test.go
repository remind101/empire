package empire

import "testing"

func TestMapRepo(t *testing.T) {
	tests := []struct {
		in  Repo
		out Repo
	}{
		{"remind101/r101-api", "remind/r101-api"},
		{"ejholmes/acme-inc", "ejholmes/acme-inc"},
	}

	for _, tt := range tests {
		out := mapRepo(tt.in)

		if got, want := out, tt.out; got != want {
			t.Errorf("mapRepo(%s) => %q; want %q", tt.in, got, want)
		}
	}
}
