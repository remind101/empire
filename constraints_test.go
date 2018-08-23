package empire

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	. "github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/pkg/constraints"
)

func TestConstraints_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		in  string
		out Constraints
		err error
	}{
		{"512:1KB", Constraints{512, 1024}, nil},
		{"512:1KB:nproc=512", Constraints{512, 1024}, nil},
		{"1025:1KB", Constraints{1025, 1024}, nil},
		{"0:1KB", Constraints{}, constraints.ErrInvalidCPUShare},

		{"1024", Constraints{}, constraints.ErrInvalidConstraint},
	}

	for i, tt := range tests {
		var c Constraints

		err := json.Unmarshal([]byte(fmt.Sprintf(`"%s"`, tt.in)), &c)
		if err != tt.err {
			t.Fatalf("#%d: err => %v; want %v", i, err, tt.err)
		}

		if tt.err != nil {
			continue
		}

		if got, want := c, tt.out; !reflect.DeepEqual(got, want) {
			t.Fatalf("#%d: Constraints => %v; want %v", i, got, want)
		}
	}
}

func TestConstraints_String(t *testing.T) {
	tests := []struct {
		in  Constraints
		out string
	}{
		// Named constraints
		{Constraints1X, "1X"},
		{Constraints2X, "2X"},
		{ConstraintsPX, "PX"},

		{Constraints{100, constraints.Memory(1 * MB)}, "100:1.00mb"},
	}

	for _, tt := range tests {
		out := tt.in.String()

		if got, want := out, tt.out; got != want {
			t.Fatalf(".String() => %s; want %s", got, want)
		}
	}
}
