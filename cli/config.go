package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	configOrder []string
	configFile  = fmt.Sprintf("%s/.emprc", os.Getenv("HOME"))
	config      = readFile(configFile)
)

func readFile(path string) map[string]string {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var config = make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.Split(scanner.Text(), "=")
		config[line[0]] = line[1]
		configOrder = append(configOrder, line[0])
	}
	return config
}
