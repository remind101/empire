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
	config      = readConfig()
)

func readConfig() map[string]string {
	file, err := os.Open(configFile)
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

func saveConfig() {
	file, err := os.Create(configFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, key := range configOrder {
		_, err := w.WriteString(key + "=" + config[key] + "\n")
		if err != nil {
			log.Fatal(err)
		}
	}

	w.Flush()
}

func deleteOrder(key string) {
	for i, v := range configOrder {
		if v == key {
			configOrder = append(configOrder[:i], configOrder[i+1:]...)
		}
	}
}
