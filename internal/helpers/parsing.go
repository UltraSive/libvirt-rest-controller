package helpers

import (
	"bufio"
	"fmt"
	"strings"
)

func ParseDomainStatus(dominfo string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(dominfo))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "State:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			status := strings.TrimSpace(parts[1])
			return status, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning output: %w", err)
	}

	return "", fmt.Errorf("status not found in domain info")
}
