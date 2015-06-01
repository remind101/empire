package util

import (
	"reflect"
	"runtime"
)

func ClassName(err error) string {
	return reflect.TypeOf(err).String()
}

func FunctionName(pc uintptr) string {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "???"
	}
	return fn.Name()
}
