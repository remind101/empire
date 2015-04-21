package logger

import (
	"log"
	"os"
)

func ExampleLogger_Log() {
	l := New(log.New(os.Stdout, "", 0))

	// Consecutive arguments are treated as key value pairs.
	l.Log("key", "value")

	// If the number of arguments is uneven, the last argument will be
	// treated as a string message.
	l.Log("key", "value", "message")

	// Output:
	// key=value
	// key=value message
}
