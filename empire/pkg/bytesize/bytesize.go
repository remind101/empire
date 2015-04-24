// package bytesize contains constants for easily switching between different
// byte sizes.
package bytesize

const (
	_       = iota // ignore first value by assigning to blank identifier
	KB uint = 1 << (10 * iota)
	MB
	GB
	TB
	PB
)
