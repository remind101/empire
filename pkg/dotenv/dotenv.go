// Package dotenv is a go port of the ruby dotenv library (https://github.com/bkeepers/dotenv)
package dotenv

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
)

// ReadFile reads parses the .env formatted file at filename and returns the
// environment variables as a map.
func ReadFile(filename string) (map[string]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return Read(file)
}

// Read parses the .env formatted io.Reader and returns the environment
// variables as a map.
func Read(r io.Reader) (map[string]string, error) {
	return read(r)
}

func read(file io.Reader) (envMap map[string]string, err error) {
	envMap = make(map[string]string)

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err = scanner.Err(); err != nil {
		return
	}

	for _, fullLine := range lines {
		if !isIgnoredLine(fullLine) {
			var key, value string
			key, value, err = parseLine(fullLine)

			if err != nil {
				return
			}
			envMap[key] = value
		}
	}
	return
}

func parseLine(line string) (key string, value string, err error) {
	if len(line) == 0 {
		err = errors.New("zero length string")
		return
	}

	// ditch the comments (but keep quoted hashes)
	if strings.Contains(line, "#") {
		segmentsBetweenHashes := strings.Split(line, "#")
		quotesAreOpen := false
		var segmentsToKeep []string
		for _, segment := range segmentsBetweenHashes {
			if strings.Count(segment, "\"") == 1 || strings.Count(segment, "'") == 1 {
				if quotesAreOpen {
					quotesAreOpen = false
					segmentsToKeep = append(segmentsToKeep, segment)
				} else {
					quotesAreOpen = true
				}
			}

			if len(segmentsToKeep) == 0 || quotesAreOpen {
				segmentsToKeep = append(segmentsToKeep, segment)
			}
		}

		line = strings.Join(segmentsToKeep, "#")
	}

	// now split key from value
	splitString := strings.SplitN(line, "=", 2)

	if len(splitString) != 2 {
		// try yaml mode!
		splitString = strings.SplitN(line, ":", 2)
	}

	if len(splitString) != 2 {
		err = errors.New("Can't separate key from value")
		return
	}

	// Parse the key
	key = splitString[0]
	if strings.HasPrefix(key, "export") {
		key = strings.TrimPrefix(key, "export")
	}
	key = strings.Trim(key, " ")

	// Parse the value
	value = splitString[1]

	// trim
	value = strings.Trim(value, " ")

	// check if we've got quoted values
	if value != "" {
		first := string(value[0:1])
		last := string(value[len(value)-1:])
		if first == last && strings.ContainsAny(first, `"'`) {
			// pull the quotes off the edges
			value = strings.Trim(value, `"'`)

			// expand quotes
			value = strings.Replace(value, `\"`, `"`, -1)
			// expand newlines
			value = strings.Replace(value, `\n`, "\n", -1)
		}
	}

	return
}

func isIgnoredLine(line string) bool {
	trimmedLine := strings.Trim(line, " \n\t")
	return len(trimmedLine) == 0 || strings.HasPrefix(trimmedLine, "#")
}
