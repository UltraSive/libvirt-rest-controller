package qemu

import (
	"encoding/json"
	"fmt"

	"libvirt-controller/internal/cmdutil"
)

func GuestPing(vm string) error {
	_, err := cmdutil.Execute("virsh", "qemu-agent-command", vm, `{"execute":"guest-ping"}`, "--pretty")
	return err
}

func GetHostName(vm string) (string, error) {
	out, err := cmdutil.Execute("virsh", "qemu-agent-command", vm, `{"execute":"guest-get-host-name"}`, "--pretty")
	if err != nil {
		return "", err
	}

	var res HostnameResponse
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return "", fmt.Errorf("failed to parse hostname response: %w", err)
	}
	return res.Return, nil
}

func GetOSInfo(vm string) (*OSInfo, error) {
	out, err := cmdutil.Execute("virsh", "qemu-agent-command", vm, `{"execute":"guest-get-osinfo"}`, "--pretty")
	if err != nil {
		return nil, err
	}

	var res OSInfoResponse
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return nil, fmt.Errorf("failed to parse OS info: %w", err)
	}
	return &res.Return, nil
}

func GetFileSystemInfo(vm string) ([]FileSystemInfo, error) {
	out, err := cmdutil.Execute("virsh", "qemu-agent-command", vm, `{"execute":"guest-get-fsinfo"}`, "--pretty")
	if err != nil {
		return nil, err
	}

	var res FSInfoResponse
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return nil, fmt.Errorf("failed to parse FS info: %w", err)
	}
	return res.Return, nil
}

func GetNetworkInterfaces(vm string) ([]NetworkInterface, error) {
	out, err := cmdutil.Execute("virsh", "qemu-agent-command", vm, `{"execute":"guest-network-get-interfaces"}`, "--pretty")
	if err != nil {
		return nil, err
	}

	var res NetInfoResponse
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return nil, fmt.Errorf("failed to parse network info: %w", err)
	}
	return res.Return, nil
}

func GetGuestTime(vm string) (*GuestTime, error) {
	out, err := cmdutil.Execute("virsh", "qemu-agent-command", vm, `{"execute":"guest-get-time"}`, "--pretty")
	if err != nil {
		return nil, err
	}

	var res TimeResponse
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return nil, fmt.Errorf("failed to parse guest time: %w", err)
	}
	return &res.Return, nil
}

func GetLoggedInUsers(vm string) ([]GuestUser, error) {
	out, err := cmdutil.Execute("virsh", "qemu-agent-command", vm, `{"execute":"guest-get-users"}`, "--pretty")
	if err != nil {
		return nil, err
	}

	var res UserResponse
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}
	return res.Return, nil
}
