package filesystem

import (
	"fmt"
	"os"
)

func CreateDirectory(path string, mode os.FileMode) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("VM directory already exists")
	}
	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("Failed to create VM directory")
	}
	return nil
}
