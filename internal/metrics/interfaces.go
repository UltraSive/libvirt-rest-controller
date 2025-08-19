package metrics

import (
	"libvirt-controller/internal/libvirt"

	"github.com/prometheus/client_golang/prometheus"
)

type LibvirtCollector struct {
	rxBytes   *prometheus.Desc
	txBytes   *prometheus.Desc
	rxPackets *prometheus.Desc
	txPackets *prometheus.Desc
}

func NewLibvirtCollector() *LibvirtCollector {
	return &LibvirtCollector{
		rxBytes: prometheus.NewDesc(
			"libvirt_domain_interface_rx_bytes_total",
			"Received bytes on a domain interface",
			[]string{"domain", "iface", "mac"},
			nil,
		),
		txBytes: prometheus.NewDesc(
			"libvirt_domain_interface_tx_bytes_total",
			"Transmitted bytes on a domain interface",
			[]string{"domain", "iface", "mac"},
			nil,
		),
		rxPackets: prometheus.NewDesc(
			"libvirt_domain_interface_rx_packets_total",
			"Received packets on a domain interface",
			[]string{"domain", "iface", "mac"},
			nil,
		),
		txPackets: prometheus.NewDesc(
			"libvirt_domain_interface_tx_packets_total",
			"Transmitted packets on a domain interface",
			[]string{"domain", "iface", "mac"},
			nil,
		),
	}
}

func (c *LibvirtCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.rxBytes
	ch <- c.txBytes
	ch <- c.rxPackets
	ch <- c.txPackets
}

func (c *LibvirtCollector) Collect(ch chan<- prometheus.Metric) {
	domains := libvirt.GetDomains()
	for _, d := range domains {
		ifaces := libvirt.GetDomainIfaces(d)
		for _, iface := range ifaces {
			stats := libvirt.GetIfaceStats(d, iface.Name)
			if stats != nil {
				ch <- prometheus.MustNewConstMetric(c.rxBytes, prometheus.CounterValue, stats["rx_bytes"], d, iface.Name, iface.Mac)
				ch <- prometheus.MustNewConstMetric(c.txBytes, prometheus.CounterValue, stats["tx_bytes"], d, iface.Name, iface.Mac)
				ch <- prometheus.MustNewConstMetric(c.rxPackets, prometheus.CounterValue, stats["rx_pkts"], d, iface.Name, iface.Mac)
				ch <- prometheus.MustNewConstMetric(c.txPackets, prometheus.CounterValue, stats["tx_pkts"], d, iface.Name, iface.Mac)
			}
		}
	}
}
