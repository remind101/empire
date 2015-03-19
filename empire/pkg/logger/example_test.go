package logger

import "os"

func ExampleLogger_Log() {
	l := New(os.Stdout, "1234")

	// Consecutive arguments are treated as key value pairs.
	l.Log("key", "value")

	// If the number of arguments is uneven, the last argument will be
	// treated as a string message.
	l.Log("key", "value", "message")

	// Output:
	// request_id=1234 key=value
	// request_id=1234 key=value message
}
