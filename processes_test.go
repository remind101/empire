package empire

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFormation(t *testing.T) {
	tests := []struct {
		f     Formation
		other Formation

		expected Formation
	}{

		// Check that default quantity and constraints are merged in.
		{
			f: Formation{
				"web": Process{
					Command: Command{"./bin/web"},
				},
				"worker": Process{
					Command: Command{"sidekiq"},
				},
			},
			other: nil,
			expected: Formation{
				"web": Process{
					Quantity: 1,
					Command:  Command{"./bin/web"},
					Memory:   NamedConstraints["1X"].Memory,
					CPUShare: NamedConstraints["1X"].CPUShare,
				},
				"worker": Process{
					Quantity: 0,
					Command:  Command{"sidekiq"},
					Memory:   NamedConstraints["1X"].Memory,
					CPUShare: NamedConstraints["1X"].CPUShare,
				},
			},
		},

		// Check that the quantity and constraints from the existing
		// configuration are used.
		{
			f: Formation{
				"web": Process{
					Command: Command{"./bin/web"},
				},
			},
			other: Formation{
				"web": Process{
					Command:  Command{"./bin/web"},
					Quantity: 2,
					Memory:   NamedConstraints["PX"].Memory,
					CPUShare: NamedConstraints["PX"].CPUShare,
					Ulimits: []Ulimit{
						{"nproc", 256, 256},
					},
				},
			},
			expected: Formation{
				"web": Process{
					Quantity: 2,
					Command:  Command{"./bin/web"},
					Memory:   NamedConstraints["PX"].Memory,
					CPUShare: NamedConstraints["PX"].CPUShare,
					Ulimits: []Ulimit{
						{"nproc", 256, 256},
					},
				},
			},
		},

		// Check that removed processes are ignored.
		{
			f: Formation{
				"web": Process{
					Command: Command{"./bin/web"},
				},
			},
			other: Formation{
				"worker": Process{
					Command:  Command{"sidekiq"},
					Quantity: 2,
					Memory:   NamedConstraints["PX"].Memory,
					CPUShare: NamedConstraints["PX"].CPUShare,
				},
			},
			expected: Formation{
				"web": Process{
					Quantity: 1,
					Command:  Command{"./bin/web"},
					Memory:   NamedConstraints["1X"].Memory,
					CPUShare: NamedConstraints["1X"].CPUShare,
				},
			},
		},
	}

	for _, tt := range tests {
		f := tt.f.Merge(tt.other)
		assert.Equal(t, tt.expected, f)
	}
}

func ExampleCommand() {
	cmd := Command{"/bin/ls", "-h"}
	fmt.Println(cmd)
}

func ExampleParseCommand() {
	cmd1, _ := ParseCommand(`/bin/ls -h`)
	cmd2, _ := ParseCommand(`/bin/echo 'hello world'`)
	fmt.Printf("%#v\n", cmd1)
	fmt.Printf("%#v\n", cmd2)
	// Output:
	// empire.Command{"/bin/ls", "-h"}
	// empire.Command{"/bin/echo", "hello world"}

}
