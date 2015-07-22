package tugboat

import (
	"errors"
	"reflect"
	"testing"
)

func TestUpdateStatus(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		fn     func() error
		status StatusUpdate
	}{
		{
			fn: func() error {
				return nil
			},
			status: StatusUpdate{
				Status: StatusSucceeded,
			},
		},
		{
			fn: func() error {
				return ErrFailed
			},
			status: StatusUpdate{
				Status: StatusFailed,
			},
		},
		{
			fn: func() error {
				return boom
			},
			status: StatusUpdate{
				Status: StatusErrored,
				Error:  &boom,
			},
		},
		{
			fn: func() error {
				panic("boom")
			},
			status: StatusUpdate{
				Status: StatusErrored,
				Error:  &boom,
			},
		},
		{
			fn: func() error {
				panic(errors.New("boom"))
			},
			status: StatusUpdate{
				Status: StatusErrored,
				Error:  &boom,
			},
		},
	}

	for i, tt := range tests {
		update := statusUpdate(tt.fn)

		if got, want := update, tt.status; !reflect.DeepEqual(got, want) {
			t.Fatalf("#%d: Status => %s; want %s", i, got, want)
		}
	}
}
