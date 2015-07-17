package headerutil

import "strings"

type Range map[string]string

func ParseRange(header string) Range {
	// version ..; max=20, , order=desc
	rangeHeader := make(Range)

	for _, i := range strings.Split(header, ",") {
		for _, j := range strings.Split(i, ";") {
			keyV := strings.Split(j, "=")
			if len(keyV) > 1 {
				rangeHeader[strings.TrimSpace(keyV[0])] = strings.TrimSpace(keyV[1])
			}
		}
	}

	return rangeHeader
}
