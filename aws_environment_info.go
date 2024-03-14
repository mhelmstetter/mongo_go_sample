package main

import (
	"bufio"
	"os"
	"strings"
)

// GetEnvironmentName extracts the environment name from the specified JSON file
func GetEnvironmentName() (string, error) {
	// Open the JSON file
	file, err := os.Open("/var/lib/cfn-init/data/metadata.json")
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Search for the line containing "environment_name"
	var environmentName string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "environment_name") {
			// Split the line by colon and take the second part
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				environmentName = strings.TrimSpace(parts[1])
				// Remove surrounding quotes, if any
				environmentName = strings.Trim(environmentName, `"`)
				break
			}
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return environmentName, nil
}
