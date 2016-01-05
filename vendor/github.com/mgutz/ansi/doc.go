/*
Small, fast library to create ANSI colored strings and codes.

Example

	// colorize a string, slowest method
	msg := ansi.Color("foo", "red+b:white")

	// create a closure to avoid escape code compilation
	phosphorize := ansi.ColorFunc("green+h:black")
	msg := phosphorize("Bring back the 80s!")

	// cache escape codes and build strings manually, faster than closure
	lime := ansi.ColorCode("green+h:black")
	reset := ansi.ColorCode("reset")

	msg := lime + "Bring back the 80s!" + reset

Other examples

	Color(s, "red")            // red
	Color(s, "red+b")          // red bold
	Color(s, "red+B")          // red blinking
	Color(s, "red+u")          // red underline
	Color(s, "red+bh")         // red bold bright
	Color(s, "red:white")      // red on white
	Color(s, "red+b:white+h")  // red bold on white bright
	Color(s, "red+B:white+h")  // red blink on white bright

To view color combinations, from terminal

	cd $GOPATH/src/github.com/mgutz/ansi
	go test

Style format

	"foregroundColor+attributes:backgroundColor+attributes"

Colors

	black
	red
	green
	yellow
	blue
	magenta
	cyan
	white

Attributes

	b = bold foreground
	B = Blink foreground
	u = underline foreground
	h = high intensity (bright) foreground, background
	i = inverse

Wikipedia ANSI escape codes [Colors](http://en.wikipedia.org/wiki/ANSI_escape_code#Colors)
*/
package ansi
