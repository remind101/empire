package main

import (
	"testing"
	"time"
)

func TestRoundDur(t *testing.T) {
	ts := []struct {
		w int
		d time.Duration
	}{
		{2, 121 * time.Second},
		{2, 149 * time.Second},
		{3, 151 * time.Second},
		{3, 179 * time.Second},
	}

	for _, ts := range ts {
		g := roundDur(ts.d, time.Minute)
		if ts.w != g {
			t.Errorf("%d != %d", g, ts.w)
		}
	}
}
