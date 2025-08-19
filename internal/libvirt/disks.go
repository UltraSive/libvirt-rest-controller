package libvirt

import (
	"fmt"
	"libvirt-controller/internal/cmdutil"
	"log"
	"strings"
)

// For Metrics
type diskInfo struct {
	Name string
}

func GetDomainDisks(domain string) []diskInfo {
	out, err := cmdutil.Execute("virsh", "domblklist", domain)
	if err != nil {
		log.Printf("error listing libvirt domain's disks")
	}
	lines := strings.Split(out, "\n")
	var disks []diskInfo
	for _, l := range lines {
		fields := strings.Fields(l)
		if len(fields) >= 2 && fields[0] != "Target" {
			disks = append(disks, diskInfo{
				Name: fields[0],
			})
		}
	}
	return disks
}

func GetDiskStats(domain, disk string) map[string]float64 {
	out, err := cmdutil.Execute("virsh", "domblkstat", domain, disk)
	if err != nil {
		log.Printf("error getting disk stats for %s", disk)
		return nil
	}
	lines := strings.Split(out, "\n")
	stats := make(map[string]float64)
	for _, l := range lines {
		fields := strings.Fields(l)
		if len(fields) == 3 {
			var val float64
			fmt.Sscanf(fields[2], "%f", &val)
			switch fields[1] {
			case "rd_bytes":
				stats["rd_bytes"] = val
			case "rd_req":
				stats["rd_req"] = val
			case "wr_bytes":
				stats["wr_bytes"] = val
			case "wr_req":
				stats["wr_req"] = val
			}
		}
	}
	return stats
}
