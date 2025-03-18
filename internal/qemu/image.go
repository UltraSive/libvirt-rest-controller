package qemu

import (
	"fmt"

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
