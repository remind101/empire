package kinesumer

import (
	"fmt"

	k "github.com/remind101/kinesumer/interface"
)

func DefaultErrHandler(err k.Error) {
	fmt.Println(err.Severity()+":", err.Error())

	severity := err.Severity()
	if severity == ECrit || severity == EError {
		panic(err)
	}
}
