package helpers

import (
	"fmt"
	"os"
	"path/filepath"

	"libvirt-controller/internal/cmdutil"
)

// ResizeDisk resizes the disk image to the desired size in GB.
func ResizeDisk(imagePath string, sizeGB int) error {
	// Convert size in GB to the required format for qemu-img (e.g., "10G" for 10 GB)
	size := fmt.Sprintf("%dG", sizeGB)

	// Use cmdutil.Execute to run the qemu-img command
	_, err := cmdutil.Execute("qemu-img", "resize", imagePath, size)
	if err != nil {
		return fmt.Errorf("failed to resize disk image: %w", err)
	}

	return nil
}

// GenerateCloudInitISO creates a cloud-init ISO, including an empty one if no files are available.
func GenerateCloudInitISO(dir string) error {
	isoPath := filepath.Join(dir, "cloud-init.iso")
	files := []string{
		filepath.Join(dir, "meta-data"),
		filepath.Join(dir, "vendor-data"),
		filepath.Join(dir, "user-data"),
		filepath.Join(dir, "network-data"),
	}

	// Filter out missing files
	var validFiles []string
	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			validFiles = append(validFiles, file)
		}
	}

	// Ensure at least one file (use /dev/null as a placeholder for an empty ISO to ensure valid libvirt XML spec)
	if len(validFiles) == 0 {
		validFiles = append(validFiles, "/dev/null")
	}

	_, err := cmdutil.Execute("genisoimage",
		append([]string{
			"-output", isoPath,
			"-volid", "cidata",
			"-joliet",
			"-rock",
		}, validFiles...)...,
	)
	if err != nil {
		return fmt.Errorf("failed to create cloud-init ISO: %w", err)
	}

	fmt.Println("Successfully created", isoPath)
	return nil
}
