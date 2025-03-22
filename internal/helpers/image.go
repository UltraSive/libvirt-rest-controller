package helpers

import (
	"fmt"
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

// GenerateCloudInitISO creates a cloud-init ISO in the specified directory.
func GenerateCloudInitISO(dir string) error {
	isoPath := filepath.Join(dir, "cloud-init.iso")
	userData := filepath.Join(dir, "user-data")
	metaData := filepath.Join(dir, "meta-data")
	networkData := filepath.Join(dir, "network-data")

	_, err := cmdutil.Execute("genisoimage",
		"-output", isoPath,
		"-volid", "cidata",
		"-joliet",
		"-rock",
		userData, metaData, networkData)
	if err != nil {
		return fmt.Errorf("failed to create cloud-init ISO: %w", err)
	}

	fmt.Println("Successfully created", isoPath)
	return nil
}
