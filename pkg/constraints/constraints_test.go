package constraints

import (
	"testing"

	. "github.com/remind101/empire/pkg/bytesize"
)

func TestCPUShare_ParseCPUShare(t *testing.T) {
	tests := []struct {
		in  string
		out CPUShare
		err error
	}{
		{"1024", 1024, nil},
		{"1025", 0, ErrInvalidCPUShare},
	}

	for _, tt := range tests {
		c, err := ParseCPUShare(tt.in)
		if err != tt.err {
			t.Fatalf("err => %v; want %v", err, tt.err)
		}

		if tt.err != nil {
			continue
		}

		if got, want := c, tt.out; got != want {
			t.Fatalf("CPUShare => %d; want %d", got, want)
		}
	}
}

func TestMemory_ParseMemory(t *testing.T) {
	tests := []struct {
		in  string
		out Memory
		err error
	}{
		{"1", 1, nil},
		{"1KB", 1024, nil},
		{"1MB", 1048576, nil},
		{"1GB", 1073741824, nil},

		{"1kB", 1024, nil},
		{"1kb", 1024, nil},
		{"1Kb", 1024, nil},

		{"", 0, ErrInvalidMemory},
		{"f", 0, ErrInvalidMemory},
		{"shitGB", 0, ErrInvalidMemory},
		{"1SHITB", 0, ErrInvalidMemory},
	}

	for i, tt := range tests {
		m, err := ParseMemory(tt.in)
		if err != tt.err {
			t.Fatalf("#%d: err => %v; want %v", i, err, tt.err)
		}

		if tt.err != nil {
			continue
		}

		if got, want := m, tt.out; got != want {
			t.Fatalf("#%d: Memory => %d; want %d", i, got, want)
		}
	}
}

func TestMemory_String(t *testing.T) {
	tests := []struct {
		in  Memory
		out string
	}{
		{1, "1"},
		{500, "500"},
		{Memory(KB), "1.00kb"},
		{Memory(2 * KB), "2.00kb"},
		{Memory(KB + 512), "1.50kb"},
		{Memory(MB), "1.00mb"},
		{Memory(2 * MB), "2.00mb"},
		{Memory(MB + (512 * KB)), "1.50mb"},
		{Memory(GB), "1.00gb"},
		{Memory(2 * GB), "2.00gb"},
		{Memory(GB + (512 * MB)), "1.50gb"},
	}

	for _, tt := range tests {
		out := tt.in.String()

		if got, want := out, tt.out; got != want {
			t.Fatalf("Memory.String() => %s; want %s", got, want)
		}
	}
}

func TestConstraints_Parse(t *testing.T) {
	tests := []struct {
		in  string
		out Constraints
		err error
	}{
		{"512:1KB", Constraints{512, 1024}, nil},
		{"1025:1KB", Constraints{}, ErrInvalidCPUShare},

		{"1024", Constraints{}, ErrInvalidConstraint},
	}

	for i, tt := range tests {
		c, err := Parse(tt.in)
		if err != tt.err {
			t.Fatalf("#%d: err => %v; want %v", i, err, tt.err)
		}

		if tt.err != nil {
			continue
		}

		if got, want := c, tt.out; got != want {
			t.Fatalf("#%d: Constraints => %v; want %v", i, got, want)
		}
	}
}
