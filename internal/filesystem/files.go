package filesystem

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// SaveFile saves data to a file in the specified directory.
func SaveFile(dir, filename string, data []byte) error {
	filePath := filepath.Join(dir, filename)
	return os.WriteFile(filePath, data, 0644) // Write raw bytes directly
}

// DeleteFile deletes a file at the specified path.
func DeleteFile(dir, filename string) error {
	filePath := filepath.Join(dir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist")
	}
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}
	return nil
}

// UpdateFile updates the content of an existing file.
func UpdateFile(dir, filename string, data []byte) error {
	filePath := filepath.Join(dir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist")
	}
	return os.WriteFile(filePath, data, 0644) // Overwrite the file with new data
}

// downloadFile handles actual downloading from the URL to a specified path
func DownloadFile(url, filePath string, mode os.FileMode) error {
	// Create the file
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// Set file permissions
	return os.Chmod(filePath, mode)
}

// downloadCachedFile manages the cache logic and uses downloadFile if necessary
func DownloadCachedFile(url, name, cacheDir string, mode os.FileMode) error {
	// Ensure cache directory exists
	err := os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		return err
	}

	// Determine the filename from the URL
	fileName := filepath.Base(url)
	cacheFilePath := filepath.Join(cacheDir, fileName)

	// Check if file is in the cache and not older than a week
	if FileExists(cacheFilePath) && !IsFileOlderThan(cacheFilePath, 7*24*time.Hour) {
		// Copy the file from cache to the destination
		return CopyFile(cacheFilePath, name, mode)
	}

	// Download the file into the cache
	err = DownloadFile(url, cacheFilePath, mode)
	if err != nil {
		return err
	}

	// Copy the cached file to the destination
	return CopyFile(cacheFilePath, name, mode)
}

// FileExists checks if a file exists at the given path
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsFileOlderThan checks if a file is older than the specified duration
func IsFileOlderThan(path string, duration time.Duration) bool {
	info, err := os.Stat(path)
	if err != nil {
		return true
	}
	return time.Since(info.ModTime()) > duration
}

// CopyFile copies a file from src to dst with the specified mode
func CopyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return os.Chmod(dst, mode)
}
