package timex

import "time"

// Now is a function that returns the current time, but can easily be stubbed
// out by setting it to a function that returns a mock value. The default is to
// call time.Now().
var Now = func() time.Time {
	return time.Now().UTC()
}
