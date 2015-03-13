package empire

import "testing"

func TestEncodeImage(t *testing.T) {
	i := Image{
		Repo: "remind101/r101-api",
		Tag:  "1234",
	}

	if got, want := encodeImage(i), "remind101/r101-api:1234"; got != want {
		t.Fatalf("encodeImage(%v) => %s; want %s", i, got, want)
	}
}

func TestDecodeImage(t *testing.T) {
	s := "remind101/r101-api:1234"
	expected := Image{
		Repo: "remind101/r101-api",
		Tag:  "1234",
	}

	if got, want := decodeImage(s), expected; got != want {
		t.Fatalf("decodeImage(%s) => %v; want %v", s, got, want)
	}
}
