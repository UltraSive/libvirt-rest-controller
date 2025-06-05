package filesystem

import (
	"fmt"
	"os"
)

// CreateDirectory creates a directory and any necessary parent directories.
// It returns nil if the directory already exists.
func CreateDirectory(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
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

// CheckDirectoryExists ensures a path points to an existing directory.
// It returns:
//   - true, nil if the path exists and is a directory.
//   - false, nil if the path does not exist.
//   - false, an error if the path exists but is not a directory, or if another error occurs.
func CheckDirectoryExists(path string) (bool, error) {
	info, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false, nil // Directory does not exist
	}
	if err != nil {
		return false, fmt.Errorf("failed to check directory status for '%s': %w", path, err)
	}

	if !info.IsDir() {
		return false, fmt.Errorf("path '%s' exists but is not a directory", path)
	}

	return true, nil // Directory exists and is a directory
}
