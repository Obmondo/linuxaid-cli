package util

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ImportOSReleaseVariables() (map[string]string, error) {
	osReleasePath := "/etc/os-release"
	file, err := os.Open(osReleasePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	osReleaseVars := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			osReleaseVars[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	return osReleaseVars, nil
}
