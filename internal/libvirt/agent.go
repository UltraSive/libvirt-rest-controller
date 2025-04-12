package libvirt

import (
	"libvirt-controller/internal/cmdutil"
	"libvirt-controller/internal/helpers"
)

// QemuAgentFileCommand executes a file command through the qemu guest agent
func QemuAgentFileCommand(domainName string, command string, path string) (
	string,
	error,
) {
	args := []string{
		"qemu-agent-command",
		domainName,
		`{"execute":"guest-file-` + command + `", "arguments":{"path":"` +
			path + `"}}`,
	}
	return cmdutil.Execute("virsh", args...)
}

// QemuAgentExec executes a command through the qemu guest agent
func QemuAgentExec(
	domainName string,
	command string,
	args []string,
	captureOutput bool,
) (string, error) {
	execArgs := []string{
		"qemu-agent-command",
		domainName,
		`{"execute":"guest-exec", "arguments":{"path":"` + command +
			`", "arg":` + helpers.ToJson(args) + `, "capture-output":` +
			helpers.ToJson(captureOutput) + `}}`,
	}
	return cmdutil.Execute("virsh", execArgs...)
}

// QemuAgentPing checks if the qemu guest agent is running
func QemuAgentPing(domainName string) (string, error) {
	return cmdutil.Execute("virsh", "qemu-agent-command", domainName,
		`{"execute":"guest-ping"}`)
}

// QemuAgentShutdown shuts down the guest OS through the qemu guest agent
func QemuAgentShutdown(domainName string, mode string) (string, error) {
	return cmdutil.Execute("virsh", "qemu-agent-command", domainName,
		`{"execute":"guest-shutdown", "arguments":{"mode":"`+mode+`"}}`)
}
