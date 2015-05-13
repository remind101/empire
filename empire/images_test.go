package empire

import "testing"

func TestEncodeImage(t *testing.T) {
	i := Image{
		Repo: "remind101/r101-api",
		ID:   "1234",
	}

	if got, want := encodeImage(i), "remind101/r101-api:1234"; got != want {
		t.Fatalf("encodeImage(%v) => %s; want %s", i, got, want)
	}
}

func TestDecodeImage(t *testing.T) {
	tests := []struct {
		in  string
		out Image
	}{
		{"remind101/r101-api:1234", Image{Repo: "remind101/r101-api", ID: "1234"}},
		{"remind101/r101-api", Image{Repo: "remind101/r101-api", ID: "latest"}},
	}

	for _, tt := range tests {
		if got, want := decodeImage(tt.in), tt.out; got != want {
			t.Fatalf("decodeImage(%s) => %v; want %v", tt.in, got, want)
		}
	}
}
