package bytesize

import "testing"

func TestSizes(t *testing.T) {
	tests := map[uint]uint{
		KB: 1024,
		MB: 1024 * 1024,
		GB: 1024 * 1024 * 1024,
		TB: 1024 * 1024 * 1024 * 1024,
		PB: 1024 * 1024 * 1024 * 1024 * 1024,
	}

	for got, want := range tests {
		if got != want {
			t.Fatalf("%v; want %v", got, want)
		}
	}
}
