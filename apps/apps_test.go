package apps

import "testing"

func TestNew(t *testing.T) {
	_, err := New("", "")
	if err != ErrInvalidName {
		t.Error("An empty name should be invalid")
	}

	a, err := New("api", "remind101/r101-api")
	if err != nil {
		t.Fatal(err)
	}

	if want, got := Name("api"), a.Name; want != got {
		t.Errorf("a.Name => %s; want %s", got, want)

	}
}
