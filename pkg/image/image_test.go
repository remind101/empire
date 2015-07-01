package image

import "testing"

var images = []struct {
	s     string
	image Image
}{
	{"ubuntu:14.04", Image{Repository: "ubuntu", Tag: "14.04"}},
	{"remind101/acme-inc", Image{Repository: "remind101/acme-inc"}},
	{"remind101/acme-inc:latest", Image{Repository: "remind101/acme-inc", Tag: "latest"}},
	{"remind101/acme-inc:foo", Image{Repository: "remind101/acme-inc", Tag: "foo"}},
	{"quay.io/remind101/acme-inc:latest", Image{Registry: "quay.io", Repository: "remind101/acme-inc", Tag: "latest"}},
	{"localhost.localdomain:5000/samalba/hipache:latest", Image{Registry: "localhost.localdomain:5000", Repository: "samalba/hipache", Tag: "latest"}},
	{"remind101/acme-inc@sha256:1234", Image{Repository: "remind101/acme-inc", Digest: "sha256:1234"}},
}

func TestDecode(t *testing.T) {
	for _, tt := range images {
		image, err := Decode(tt.s)
		if err != nil {
			t.Logf("Decode(%q)", tt.s)
			t.Fatal(err)
		}

		if got, want := image, tt.image; got != want {
			t.Logf("Decode(%q)", tt.s)
			t.Fatalf("Image => %#v; want %#v", got, want)
		}
	}
}

func TestEncode(t *testing.T) {
	for _, tt := range images {
		s := Encode(tt.image)

		if got, want := s, tt.s; got != want {
			t.Logf("Encode(%#v)", tt.image)
			t.Fatalf("Image => %v; want %v", got, want)
		}
	}
}
