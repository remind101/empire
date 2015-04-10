// package plugin is a small framework for building Go binaries that contain
// plugins for the heroku hk command https://github.com/heroku/hk.
package plugin

import (
	"fmt"
	"log"
	"os"

	"github.com/bgentry/heroku-go"
	"github.com/mgutz/ansi"
)

// A plugin represents an individual plugin.
type Plugin struct {
	// The name of the plugin.
	Name string

	// The action that will be performed when this plugin is invoked.
	Action func(*Context)
}

func Must(err error) {
	if err != nil {
		if herror, ok := err.(heroku.Error); ok {
			switch herror.Id {
			case "two_factor":
				printError(err.Error() + " Authorize with `hk authorize`.")
				os.Exit(79)
			case "unauthorized":
				printFatal(err.Error() + " Log in with `hk login`.")
			}
		}
		printFatal(err.Error())
	}
}

func printError(message string, args ...interface{}) {
	log.Println(colorizeMessage("red", "error:", message, args...))
}

func printFatal(message string, args ...interface{}) {
	log.Fatal(colorizeMessage("red", "error:", message, args...))
}

func colorizeMessage(color, prefix, message string, args ...interface{}) string {
	prefResult := ""
	if prefix != "" {
		prefResult = ansi.Color(prefix, color+"+b") + " " + ansi.ColorCode("reset")
	}
	return prefResult + ansi.Color(fmt.Sprintf(message, args...), color) + ansi.ColorCode("reset")
}
