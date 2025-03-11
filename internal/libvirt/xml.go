package libvirt

import "fmt"

// GenerateLibvirtXML creates an XML configuration for the VM
func GenerateLibvirtXML(id string, memoryMB int, cpus int) string {
	return fmt.Sprintf(`
<domain type='kvm'>
    <name>%s</name>
    <memory unit='MB'>%d</memory>
    <vcpu>%d</vcpu>
    <os>
        <type arch='x86_64'>hvm</type>
    </os>
    <devices>
        <disk type='file' device='disk'>
            <driver name='qemu' type='qcow2'/>
            <source file='/home/sive/vm/%s/disk.qcow2'/>
            <target dev='vda' bus='virtio'/>
        </disk>
        <interface type='network'>
            <source network='default'/>
        </interface>
    </devices>
</domain>`, id, memoryMB, cpus, id)
}
