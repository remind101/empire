package base62

import "testing"

func Test_Encode(t *testing.T) {
	tests := []struct {
		Input    uint64
		Expected string
	}{
		{1024, "gw"},
		{512347, "29hF"},
		{5369806, "mwVM"},
		{2147483647, "2lkCB1"},
		{0xFFFFFFFFFFFFFFFF, "lYGhA16ahyf"}, // Max uint64
	}

	for i, test := range tests {
		e := Encode(test.Input)

		if e != test.Expected {
			t.Errorf("%v: Got %v, want %v", i, e, test.Expected)
		}
	}
}
