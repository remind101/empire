package headerutil

import "testing"

func TestParseRange(t *testing.T) {
	var (
		version = "version"
		max     = 20
		order   = "desc"
	)

	tests := []struct {
		in  string
		out *Range
	}{
		{
			"version ..; max=20, , order=desc",
			&Range{
				Sort:  &version,
				Max:   &max,
				Order: &order,
			},
		},
		{
			"order=desc, version ..; max=20",
			&Range{
				Sort:  &version,
				Max:   &max,
				Order: &order,
			},
		},
		{
			", , order=desc, version ..; max=20",
			&Range{
				Sort:  &version,
				Max:   &max,
				Order: &order,
			},
		},
		{
			" , , order=desc, version ..; max=20",
			&Range{
				Sort:  &version,
				Max:   &max,
				Order: &order,
			},
		},
	}

	for i, tt := range tests {
		r, _ := ParseRange(tt.in)

		if got, want := *r.Sort, *tt.out.Sort; got != want {
			t.Fatalf("#%d: Range.Sort => %s; want %s", i, got, want)
		}

		if got, want := *r.Max, *tt.out.Max; got != want {
			t.Fatalf("#%d: Range.Max => %d; want %d", i, got, want)
		}

		if got, want := *r.Order, *tt.out.Order; got != want {
			t.Fatalf("#%d: Range.Order => %s; want %s", i, got, want)
		}
	}
}
