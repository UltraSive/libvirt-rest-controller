package libvirt

import (
	"fmt"
	"libvirt-controller/internal/cmdutil"
	"log"
	"strings"
)

// For Metrics
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

type ifaceInfo struct {
	Name string
	Mac  string
}

func GetDomainIfaces(domain string) []ifaceInfo {
	out, err := cmdutil.Execute("virsh", "domiflist", domain)
	if err != nil {
		log.Printf("error listing libvirt domain's interfaces")
	}
	lines := strings.Split(out, "\n")
	var ifaces []ifaceInfo
	for _, l := range lines {
		fields := strings.Fields(l)
		if len(fields) >= 5 && fields[0] != "Interface" {
			ifaces = append(ifaces, ifaceInfo{
				Name: fields[0],
				Mac:  fields[4],
			})
		}
	}
	return ifaces
}

func GetIfaceStats(domain, iface string) map[string]float64 {
	out, err := cmdutil.Execute("virsh", "domifstat", domain, iface)
	if err != nil {
		log.Printf("error getting interface stats")
	}
	lines := strings.Split(out, "\n")
	stats := make(map[string]float64)
	for _, l := range lines {
		fields := strings.Fields(l)
		if len(fields) == 3 {
			var val float64
			fmt.Sscanf(fields[2], "%f", &val)
			switch fields[1] {
			case "rx_bytes":
				stats["rx_bytes"] = val
			case "tx_bytes":
				stats["tx_bytes"] = val
			case "rx_packets":
				stats["rx_pkts"] = val
			case "tx_packets":
				stats["tx_pkts"] = val
			}
		}
	}
	return stats
}
