package cli

import (
	"errors"
	"strconv"
	"strings"
)

var errInvalidScaleArg = errors.New("invalid argument")

// This is a direct copy paste from
// https://github.com/remind101/emp/blob/6fc34ef0633f5a26217023ede86b866bf4f17cc4/scale.go#L127-L168.
//
// TODO: Actually write some tests for this and make it less crazy.
func parseScaleArg(arg string) (pstype string, qty int, size string, err error) {
	qty = -1
	iEquals := strings.IndexRune(arg, '=')
	if fields := strings.Fields(arg); len(fields) > 1 || iEquals == -1 {
		err = errInvalidScaleArg
		return
	}
	pstype = arg[:iEquals]

	rem := strings.ToUpper(arg[iEquals+1:])
	if len(rem) == 0 {
		err = errInvalidScaleArg
		return
	}

	if iColon := strings.IndexRune(rem, ':'); iColon == -1 {
		if iX := strings.IndexRune(rem, 'X'); iX == -1 {
			qty, err = strconv.Atoi(rem)
			if err != nil {
				return pstype, -1, "", errInvalidScaleArg
			}
		} else {
			size = rem
		}
	} else {
		if iColon > 0 {
			qty, err = strconv.Atoi(rem[:iColon])
			if err != nil {
				return pstype, -1, "", errInvalidScaleArg
			}
		}
		if len(rem) > iColon+1 {
			size = rem[iColon+1:]
		}
	}
	if err != nil || qty == -1 && size == "" {
		err = errInvalidScaleArg
	}
	return
}
