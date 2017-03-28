package main

import (
	"reflect"
	"testing"
)

func qty(v int) *int {
	return &v
}

var parseScaleTests = []struct {
	in     string
	pstype string
	qty    *int
	size   string
	err    error
}{
	{"web=5", "web", qty(5), "", nil},
	{"bg_worker=50:1X", "bg_worker", qty(50), "1X", nil},
	{"bg_worker=50:PX", "bg_worker", qty(50), "PX", nil},
	{"web=:2X", "web", nil, "2X", nil},
	{"web=:PX", "web", nil, "PX", nil},
	{"web=1X", "web", nil, "1X", nil},
	{"web=1x", "web", nil, "1X", nil},
	{"web=PX", "web", nil, "PX", nil},
	{"web=px", "web", nil, "PX", nil},
	{"worker=-1", "worker", qty(-1), "", nil},
	{"web=1X:5", "web", nil, "", errInvalidScaleArg},
	{"web=PX:5", "web", nil, "", errInvalidScaleArg},
	{"web", "", nil, "", errInvalidScaleArg},
	{"web=", "web", nil, "", errInvalidScaleArg},
	{"web =", "", nil, "", errInvalidScaleArg},
	{"web=1X: 5", "", nil, "", errInvalidScaleArg},
}

func TestParseScaleArg(t *testing.T) {
	for i, pt := range parseScaleTests {
		pstype, qty, size, err := parseScaleArg(pt.in)
		if pstype != pt.pstype {
			t.Errorf("%d. parseScaleArg(%q).pstype => %q, want %q", i, pt.in, pstype, pt.pstype)
		}
		if !reflect.DeepEqual(qty, pt.qty) {
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
