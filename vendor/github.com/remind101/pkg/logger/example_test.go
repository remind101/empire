package logger

import (
	"log"
	"os"
)

func ExampleLogger_Log() {
	l := New(log.New(os.Stdout, "", 0))

	// Consecutive arguments after the message are treated as key value pairs.
	l.Info("message", "key", "value")

	// Output:
	// message key=value
}
