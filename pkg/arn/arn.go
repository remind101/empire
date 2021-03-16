// package arn is a Go package for parsing Amazon Resource Names.
package arn

import (
	"errors"
	"strings"
)

var (
	// ErrInvalidARN may be returned when parsing a string that is not a valid ARN.
	ErrInvalidARN = errors.New("invalid ARN")

	// ErrInvalidResource may be returned when an ARN Resrouce is not valid.
	ErrInvalidResource = errors.New("invalid ARN resource")
)

const delimiter = ":"

// ARN represents a parsed Amazon Resource Name.
type ARN struct {
	ARN      string
	AWS      string
	Service  string
	Region   string
	Account  string
	Resource string
}

// Parse parses an Amazon Resource Name from a String into an ARN.
func Parse(arn string) (*ARN, error) {
	p := strings.SplitN(arn, delimiter, 6)

	// Ensure that we have all the components that make up an ARN.
	if len(p) < 6 {
		return nil, ErrInvalidARN
	}

	a := &ARN{
		ARN:      p[0],
		AWS:      p[1],
		Service:  p[2],
		Region:   p[3],
		Account:  p[4],
		Resource: p[5],
	}

	// ARN's always start with "arn:aws" (hopefully).
	if a.ARN != "arn" || a.AWS != "aws" {
		return nil, ErrInvalidARN
	}

	return a, nil
}

// String returns the string representation of an Amazon Resource Name.
func (a *ARN) String() string {
	return strings.Join(
		[]string{a.ARN, a.AWS, a.Service, a.Region, a.Account, a.Resource},
		delimiter,
	)
}

// SplitResource splits the Resource section of an ARN into its type and id
// components.
func SplitResource(r string) (resource, id string, err error) {
	p := strings.SplitN(r, "/", 2)

	if len(p) != 2 {
		err = ErrInvalidResource
		return
	}

	resource = p[0]
	id = p[1]

	return
}

// ResourceID takes an ARN string and returns the resource ID from it.
func ResourceID(arn string) (string, error) {
	a, err := Parse(arn)
	if err != nil {
		return "", err
	}

	_, id, err := SplitResource(a.Resource)
	return id, err
}
