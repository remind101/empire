package image

import "testing"

var images = []struct {
	s     string
	image Image
}{
	{"ubuntu:14.04", Image{Repository: "ubuntu", Tag: "14.04"}},
	{"remind101/acme-inc:latest", Image{Repository: "remind101/acme-inc", Tag: "latest"}},
	{"remind101/acme-inc:foo", Image{Repository: "remind101/acme-inc", Tag: "foo"}},

	// TODO
	//{"quay.io/remind101/acme-inc:foo", Image{Registry: "quay.io", Repository: "remind101/acme-inc", Tag: "foo"}},
	//{"remind101/acme-inc@sha256:1234", Image{Repository: "remind101/acme-inc", Digest: "sha256:1234"}
}

func TestDecode(t *testing.T) {
	for _, tt := range images {
		image, err := Decode(tt.s)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := image, tt.image; got != want {
			t.Fatalf("Image => %v; want %v", got, want)
		}
	}
}

func TestEncode(t *testing.T) {
	for _, tt := range images {
		s := Encode(tt.image)

		if got, want := s, tt.s; got != want {
			t.Fatalf("Image => %v; want %v", got, want)
		}
	}
}
