package libvirt

import (
	"log"
	"net"
	"sync"

	"github.com/digitalocean/go-libvirt"
)

var (
	conn *libvirt.Libvirt
	once sync.Once
	err  error
)

// GetConnection ensures only one connection is established
func GetConnection() (*libvirt.Libvirt, error) {
	once.Do(func() {
		// Open a UNIX socket to libvirt
		socket, err := net.Dial("unix", "/var/run/libvirt/libvirt-sock")
		if err != nil {
			log.Fatalf("Failed to connect to libvirt socket: %v", err)
		}

		conn = libvirt.New(socket)
		if err := conn.Connect(); err != nil {
			log.Fatalf("Failed to establish libvirt connection: %v", err)
		}
	})
	return conn, err
}
