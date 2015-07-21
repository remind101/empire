package headerutil

import (
	"strconv"
	"strings"
)

type Range struct {
	// If provided, specifies field to sort by.
	Sort *string

	// If provided, limits the results to the provided value.
	Max *int

	// The order the results are returned in.
	Order *string
}

func ParseRange(header string) (*Range, error) {
	rangeHeader := Range{}

	for _, i := range strings.Split(header, ",") {
		for _, j := range strings.Split(i, ";") {
			parts := strings.Split(j, "=")
			if len(parts) == 1 {
				if rangeHeader.Sort == nil {
					if len(parts[0]) > 0 && parts[0] != " " {
						sort := strings.TrimSpace(strings.TrimRight(parts[0], " .."))
						rangeHeader.Sort = &sort
					}
				}
			} else {
				if value := strings.TrimSpace(parts[0]); value == "max" {
					max, err := strconv.Atoi(strings.TrimSpace(parts[1]))
					if err != nil {
						return nil, err
					}
					rangeHeader.Max = &max
				} else {
					order := strings.TrimSpace(parts[1])
					rangeHeader.Order = &order
				}
			}
		}
	}

	return &rangeHeader, nil
}
