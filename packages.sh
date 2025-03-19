apt install bridge-utils qemu-kvm qemu-utils libvirt-clients libvirt-daemon-system virtinst genisoimage -y
sudo setfacl -m user:$USER:rw /var/run/libvirt/libvirt-sock