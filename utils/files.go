package utils

import (
	"bufio"
	"os"
	"regexp"
)

// The regex values.

var spaceRegex = regexp.MustCompile(`\s`)

// ImportKeys
//	Reads the keys.txt file and returns the API keys.
func ImportKeys(path string) ([]string, error) {
	// Open handle.
	readFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// New scanner.
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	// Read lines.
	keys := make([]string, 0)
	for fileScanner.Scan() {
		// Check if it's a valid key.
		key := fileScanner.Text()
		if !spaceRegex.MatchString(key) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}
