package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	hkuser, hkpass := os.Getenv("HKUSER"), os.Getenv("HKPASS")
	apiURL := os.Getenv("HEROKU_API_URL")
	repo, id := strings.Split(os.Args[1], ":")[0], strings.Split(os.Args[1], ":")[1]

	flags := []string{
		"-sS",
		"--fail",
		"-D",
		"-",
		"--user",
		fmt.Sprintf("'%s:%s'", hkuser, hkpass),
		"-X POST",
		fmt.Sprintf("%s/deploys", apiURL),
		fmt.Sprint("-d '{\"image\":{\"repo\":\"%s\",\"id\":\"%s\"}}'", repo, id),
		"-H 'Accept: application/vnd.heroku+json; version=3'",
	}

	cmd := exec.Command("curl", flags...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to deploy %s:%s\n", repo, id)
		fmt.Printf("== DEBUG: REPO:%s ID:%s USER:%s API:%s\n", repo, id, hkuser, apiURL)
		fmt.Printf("%s\n", err)
		if len(output) > 0 {
			fmt.Printf("%s", string(output))
		}
		os.Exit(1)
	}

	fmt.Printf("Deployed %s:%s\n", repo, id)
}
