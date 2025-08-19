package metrics

import (
	"libvirt-controller/internal/libvirt"

	"github.com/prometheus/client_golang/prometheus"
)

type LibvirtDiskCollector struct {
	rdBytes prometheus.Desc
	wrBytes prometheus.Desc
	rdReqs  prometheus.Desc
	wrReqs  prometheus.Desc
}

func NewLibvirtDiskCollector() *LibvirtDiskCollector {
	return &LibvirtDiskCollector{
		rdBytes: *prometheus.NewDesc("libvirt_domain_disk_read_bytes_total", "Read bytes on a domain disk", []string{"domain", "disk"}, nil),
		wrBytes: *prometheus.NewDesc("libvirt_domain_disk_write_bytes_total", "Written bytes on a domain disk", []string{"domain", "disk"}, nil),
		rdReqs:  *prometheus.NewDesc("libvirt_domain_disk_read_requests_total", "Read requests on a domain disk", []string{"domain", "disk"}, nil),
		wrReqs:  *prometheus.NewDesc("libvirt_domain_disk_write_requests_total", "Write requests on a domain disk", []string{"domain", "disk"}, nil),
	}
}

func (c *LibvirtDiskCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- &c.rdBytes
	ch <- &c.wrBytes
	ch <- &c.rdReqs
	ch <- &c.wrReqs
}

func (c *LibvirtDiskCollector) Collect(ch chan<- prometheus.Metric) {
	domains := libvirt.GetDomains()
	for _, d := range domains {
		disks := libvirt.GetDomainDisks(d)
		for _, disk := range disks {
			stats := libvirt.GetDiskStats(d, disk.Name)
			if stats != nil {
				ch <- prometheus.MustNewConstMetric(&c.rdBytes, prometheus.CounterValue, stats["rd_bytes"], d, disk.Name)
				ch <- prometheus.MustNewConstMetric(&c.wrBytes, prometheus.CounterValue, stats["wr_bytes"], d, disk.Name)
				ch <- prometheus.MustNewConstMetric(&c.rdReqs, prometheus.CounterValue, stats["rd_req"], d, disk.Name)
				ch <- prometheus.MustNewConstMetric(&c.wrReqs, prometheus.CounterValue, stats["wr_req"], d, disk.Name)
			}
		}
	}
}
