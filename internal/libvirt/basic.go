package libvirt

import (
	"libvirt-controller/internal/cmdutil"
	"log"
	"strings"
)

func GetDomains() []string {
	out, err := cmdutil.Execute("virsh", "list", "--name")
	if err != nil {
		log.Printf("error listing libvirt domains")
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var domains []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			domains = append(domains, l)
		}
	}
	return domains
}

// DefineDomain defines a domain from an XML file
func DefineDomain(xmlConfigPath string) (string, error) {
	return cmdutil.Execute("virsh", "define", xmlConfigPath)
}

func UndefineDomain(domainName string) (string, error) {
	return cmdutil.Execute("virsh", "undefine", domainName)
}

func StartDomain(domainName string) (string, error) {
	return cmdutil.Execute("virsh", "start", domainName)
}

func RebootDomain(domainName string) (string, error) {
	return cmdutil.Execute("virsh", "reboot", domainName)
}

func ResetDomain(domainName string) (string, error) {
	return cmdutil.Execute("virsh", "reset", domainName)
}

func ShutdownDomain(domainName string) (string, error) {
	return cmdutil.Execute("virsh", "shutdown", domainName)
}

func DestroyDomain(domainName string) (string, error) {
	return cmdutil.Execute("virsh", "destroy", domainName)
}

func SuspendDomain(domainName string) (string, error) {
	return cmdutil.Execute("virsh", "suspend", domainName)
}

func ResumeDomain(domainName string) (string, error) {
	return cmdutil.Execute("virsh", "resume", domainName)
}

func GetDomainInfo(domainName string) (string, error) {
	return cmdutil.Execute("virsh", "dominfo", domainName)
}
