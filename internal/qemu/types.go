package qemu

type HostnameResponse struct {
	Return string `json:"return"`
}

type OSInfo struct {
	Name          string `json:"name"`
	KernelRelease string `json:"kernel-release"`
	Version       string `json:"version"`
	PrettyName    string `json:"pretty-name"`
	KernelVersion string `json:"kernel-version"`
	ID            string `json:"id"`
}

type OSInfoResponse struct {
	Return OSInfo `json:"return"`
}

type FileSystemInfo struct {
	Name              string `json:"name"`
	Mountpoint        string `json:"mountpoint"`
	FilesystemType    string `json:"filesystem-type"`
	LogicalBlockSize  int    `json:"logical-block-size"`
	PhysicalBlockSize int    `json:"physical-block-size"`
}

type FSInfoResponse struct {
	Return []FileSystemInfo `json:"return"`
}

type NetworkInterface struct {
	Name            string `json:"name"`
	HardwareAddress string `json:"hardware-address"`
	IPAddresses     []struct {
		IPAddress     string `json:"ip-address"`
		Prefix        int    `json:"prefix"`
		IPAddressType string `json:"ip-address-type"`
	} `json:"ip-addresses"`
}

type NetInfoResponse struct {
	Return []NetworkInterface `json:"return"`
}

type GuestTime struct {
	Seconds     int64 `json:"seconds"`
	Nanoseconds int64 `json:"nanoseconds"`
}

type TimeResponse struct {
	Return GuestTime `json:"return"`
}

type GuestUser struct {
	User      string `json:"user"`
	Domain    string `json:"domain"`
	LoginTime int64  `json:"login-time"`
	UserID    int    `json:"user-id"`
}

type UserResponse struct {
	Return []GuestUser `json:"return"`
}
