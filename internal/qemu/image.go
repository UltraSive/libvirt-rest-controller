package qemu

import (
	"fmt"
	"os/exec"
)

// ResizeDisk resizes the disk image to the desired size in GB.
func ResizeDisk(imagePath string, sizeGB int) error {
	// Convert size in GB to the required format for qemu-img (e.g., "10G" for 10 GB)
	size := fmt.Sprintf("%dG", sizeGB)

	// Construct the qemu-img command to resize the disk
	cmd := exec.Command("qemu-img", "resize", imagePath, size)

	// Run the command
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to resize disk image: %w", err)
	}

	return nil
}
