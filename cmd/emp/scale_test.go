package main

import (
	"testing"
)

var parseScaleTests = []struct {
	in     string
	pstype string
	qty    int
	size   string
	err    error
}{
	{"web=5", "web", 5, "", nil},
	{"bg_worker=50:1X", "bg_worker", 50, "1X", nil},
	{"bg_worker=50:PX", "bg_worker", 50, "PX", nil},
	{"web=:2X", "web", -1, "2X", nil},
	{"web=:PX", "web", -1, "PX", nil},
	{"web=1X", "web", -1, "1X", nil},
	{"web=1x", "web", -1, "1X", nil},
	{"web=PX", "web", -1, "PX", nil},
	{"web=px", "web", -1, "PX", nil},
	{"web=1X:5", "web", -1, "", errInvalidScaleArg},
	{"web=PX:5", "web", -1, "", errInvalidScaleArg},
	{"web", "", -1, "", errInvalidScaleArg},
	{"web=", "web", -1, "", errInvalidScaleArg},
	{"web =", "", -1, "", errInvalidScaleArg},
	{"web=1X: 5", "", -1, "", errInvalidScaleArg},
}

func TestParseScaleArg(t *testing.T) {
	for i, pt := range parseScaleTests {
		pstype, qty, size, err := parseScaleArg(pt.in)
		if pstype != pt.pstype {
			t.Errorf("%d. parseScaleArg(%q).pstype => %q, want %q", i, pt.in, pstype, pt.pstype)
		}
		if qty != pt.qty {
			t.Errorf("%d. parseScaleArg(%q).qty => %d, want %d", i, pt.in, qty, pt.qty)
		}
		if size != pt.size {
			t.Errorf("%d. parseScaleArg(%q).size => %q, want %q", i, pt.in, size, pt.size)
		}
		if err != pt.err {
			t.Errorf("%d. parseScaleArg(%q).err => %q, want %q", i, pt.in, err, pt.err)
		}
	}
}
