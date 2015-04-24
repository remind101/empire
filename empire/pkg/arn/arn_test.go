package arn

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		in  string
		out ARN
	}{
		{"arn:aws:ecs:us-east-1:249285743859:service/acme-inc--web", ARN{
			ARN:      "arn",
			AWS:      "aws",
			Service:  "ecs",
			Region:   "us-east-1",
			Account:  "249285743859",
			Resource: "service/acme-inc--web",
		}},
		{"arn:aws:ecs:us-east-1:249285743859:service/acme-inc:web", ARN{
			ARN:      "arn",
			AWS:      "aws",
			Service:  "ecs",
			Region:   "us-east-1",
			Account:  "249285743859",
			Resource: "service/acme-inc:web",
		}},
	}

	for i, tt := range tests {
		arn, _ := Parse(tt.in)

		if got, want := arn.String(), tt.out.String(); got != want {
			t.Errorf("#%d: Parse(%q) => %s; want %s", i, tt.in, got, want)
		}
	}
}

func TestSplitResource(t *testing.T) {
	tests := []struct {
		in           string
		resource, id string
		err          error
	}{
		{"service/acme-inc", "service", "acme-inc", nil},
		{"service", "", "", ErrInvalidResource},
	}

	for i, tt := range tests {
		r, id, err := SplitResource(tt.in)
		if err != tt.err {
			t.Fatalf("#%d: SplitResource(%q): err => %v; want %v", i, tt.in, err, tt.err)
		}

		if got, want := r, tt.resource; got != want {
			t.Errorf("#%d: SplitResource(%q): resource => %v; want %v", i, tt.in, got, want)
		}

		if got, want := id, tt.id; got != want {
			t.Errorf("#%d: SplitResource(%q): id => %v; want %v", i, tt.in, got, want)
		}
	}
}
