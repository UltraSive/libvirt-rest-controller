package cmdutil

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Execute runs a command and returns the output or an error.
func Execute(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %s, %w", stderr.String(), err)
	}
	return out.String(), nil
}
