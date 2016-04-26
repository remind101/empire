package empire

import (
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
					Nproc:    NamedConstraints["1X"].Nproc,
				},
				"worker": Process{
					Quantity: 0,
					Command:  Command{"sidekiq"},
					Memory:   NamedConstraints["1X"].Memory,
					CPUShare: NamedConstraints["1X"].CPUShare,
					Nproc:    NamedConstraints["1X"].Nproc,
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
					Nproc:    NamedConstraints["PX"].Nproc,
				},
			},
			expected: Formation{
				"web": Process{
					Quantity: 2,
					Command:  Command{"./bin/web"},
					Memory:   NamedConstraints["PX"].Memory,
					CPUShare: NamedConstraints["PX"].CPUShare,
					Nproc:    NamedConstraints["PX"].Nproc,
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
					Nproc:    NamedConstraints["PX"].Nproc,
				},
			},
			expected: Formation{
				"web": Process{
					Quantity: 1,
					Command:  Command{"./bin/web"},
					Memory:   NamedConstraints["1X"].Memory,
					CPUShare: NamedConstraints["1X"].CPUShare,
					Nproc:    NamedConstraints["1X"].Nproc,
				},
			},
		},
	}

	for _, tt := range tests {
		f := tt.f.Merge(tt.other)
		assert.Equal(t, tt.expected, f)
	}
}
