package main

import (
	"fmt"
	"os"
	"os/exec"
)

func returnOutput(outs []byte) {
	if len(outs) > 0 {
		fmt.Printf("%s", string(outs))
	}
}

func main() {
	args := os.Args[1:]

	EMPIRE_URL := os.Getenv("EMPIRE_URL")
	if EMPIRE_URL == "" {
		EMPIRE_URL = "http://0.0.0.0:8080"
	}
	os.Setenv("HEROKU_API_URL", EMPIRE_URL)

	cmd := exec.Command("hk", args...)
	output, _ := cmd.CombinedOutput()
	returnOutput(output)
}
