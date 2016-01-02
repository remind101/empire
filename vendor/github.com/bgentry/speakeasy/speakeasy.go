package speakeasy

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Ask the user to enter a password with input hidden. prompt is a string to
// display before the user's input. Returns the provided password, or an error
// if the command failed.
func Ask(prompt string) (password string, err error) {
	if prompt != "" {
		fmt.Fprint(os.Stdout, prompt) // Display the prompt.
	}
	return getPassword()
}

func readline() (value string, err error) {
	var valb []byte
	var n int
	b := make([]byte, 1)
	for {
		// read one byte at a time so we don't accidentally read extra bytes
		n, err = os.Stdin.Read(b)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 || b[0] == '\n' {
			break
		}
		valb = append(valb, b[0])
	}

	// Carriage return after the user input.
	fmt.Println("")
	return strings.TrimSuffix(string(valb), "\r"), nil
}
