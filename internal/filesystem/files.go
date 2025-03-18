package filesystem

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func SaveFile(dir, filename string, data []byte) error {
	filePath := filepath.Join(dir, filename)
	return os.WriteFile(filePath, data, 0644) // Write raw bytes directly
}

// DownloadWebFile downloads a file from a given URL and saves it to a specified path
func DownloadWebFile(url, name string, mode os.FileMode) error {
	// Create the file
	out, err := os.Create(name)
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

	// Look up the qemu user
	/*qemuUser, err := user.Lookup("qemu")
	if err != nil {
		return fmt.Errorf("failed to lookup user 'qemu': %v", err)
	}*/

	// Convert UID and GID to integers
	/*uid := 64055 //strconv.Atoi(qemuUser.Uid)
	if err != nil {
		return fmt.Errorf("failed to convert UID: %v", err)
	}
	gid := 994 //strconv.Atoi(qemuUser.Gid)
	if err != nil {
		return fmt.Errorf("failed to convert GID: %v", err)
	}*/

	// Set the file's UID and GID to qemu
	/*if err := os.Chown(name, uid, gid); err != nil {
		return fmt.Errorf("failed to change file ownership: %v", err)
	}*/

	// Set file permissions
	err = os.Chmod(name, mode)
	if err != nil {
		return fmt.Errorf("failed to change file permissions: %v", err)
	}

	return nil
}
