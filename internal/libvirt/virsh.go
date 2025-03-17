package libvirt

import (
	"bytes"
	"fmt"
	"os/exec"
)

// ExecuteCommand constructs and executes the virsh command
func ExecuteCommand(args ...string) (string, error) {
	cmd := exec.Command("virsh", args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %s, %w", stderr.String(), err)
	}
	return out.String(), nil
}

// DefineDomain defines a domain from an XML file
func DefineDomain(xmlConfigPath string) (string, error) {
	return ExecuteCommand("define", xmlConfigPath)
}

// StartDomain starts a domain
func StartDomain(domainName string) (string, error) {
	return ExecuteCommand("start", domainName)
}

// StopDomain shuts down a domain
func StopDomain(domainName string) (string, error) {
	return ExecuteCommand("shutdown", domainName)
}
