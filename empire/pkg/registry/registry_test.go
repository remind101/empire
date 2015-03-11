package registry

import "testing"

func TestSplitRepo(t *testing.T) {
	tests := []struct {
		in string

		registry string
		path     string

		err error
	}{
		{"ejholmes/acme-inc", "", "ejholmes/acme-inc", nil},
		{"quay.io/ejholmes/acme-inc", "quay.io", "ejholmes/acme-inc", nil},
		{"ejholmes", "", "", ErrInvalidRepo},
	}

	for i, tt := range tests {
		registry, path, err := Split(tt.in)
		if err != tt.err {
			t.Fatalf("#%d: Error => %v; want %v", i, err, tt.err)
		}

		if got, want := registry, tt.registry; got != want {
			t.Fatalf("#%d: Registry => %s; want %s", i, got, want)
		}

		if got, want := path, tt.path; got != want {
			t.Fatalf("#%d: Path => %s; want %s", i, got, want)
		}
	}
}
