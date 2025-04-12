package libvirt

import (
	"libvirt-controller/internal/cmdutil"
)

// TakeSnapshot creates a snapshot of a VM.
// quiesce:  If true, attempt to quiesce the guest filesystem before taking the snapshot.
func TakeSnapshot(domainName string, snapshotName string, quiesce bool) (string, error) {
	cmd := []string{
		"snapshot-create-as",
		domainName,
		snapshotName,
		//"--disk-only",   // create snapshot of disk only (avoid memory snapshot)
		//"--no-metadata", // skip saving metadata
	}

	if quiesce {
		cmd = append(cmd, "--quiesce")
	}

	return cmdutil.Execute("virsh", cmd...)
}

// RevertSnapshot reverts the VM's disk to the state of the snapshot and deletes the snapshot.
func RevertSnapshot(domainName string, snapshotName string) (string, error) {
	cmd := []string{
		"snapshot-revert",
		domainName,
		snapshotName,
		//"--disk-only",
	}

	return cmdutil.Execute("virsh", cmd...)
}

// DeleteSnapshot deletes a snapshot.
// Essentially commits changes made since the snapshot was taken.
func DeleteSnapshot(domainName string, snapshotName string) (string, error) {
	cmd := []string{
		"snapshot-delete",
		domainName,
		snapshotName,
		"--metadata",
	}
	return cmdutil.Execute("virsh", cmd...)
}
