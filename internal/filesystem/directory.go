package filesystem

import (
	"fmt"
	"os"
)

// CreateDirectory creates a new directory with the specified mode.
func CreateDirectory(path string, mode os.FileMode) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("VM directory already exists")
	}
	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("Failed to create VM directory")
	}
	return nil
}

// DeleteDirectory removes a directory at the specified path.
func DeleteDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist")
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to delete directory: %v", err)
	}
	return nil
}
