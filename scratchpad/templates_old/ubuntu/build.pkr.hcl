# Build Definition to create the VM Template
build {

  name = "ubuntu-server"
  sources = [
     "source.proxmox-iso.default",
  ]

  # Provisioning the VM Template for Cloud-Init Integration in Proxmox #1
  provisioner "shell" {
    inline = [
      "while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done",
      "sudo rm /etc/ssh/ssh_host_*",
      "sudo truncate -s 0 /etc/machine-id",
      "sudo apt -y autoremove --purge",
      "sudo apt -y clean",
      "sudo apt -y autoclean",
      "sudo cloud-init clean",
      "sudo rm -f /etc/cloud/cloud.cfg.d/subiquity-disable-cloudinit-networking.cfg",
      "sudo rm -f /etc/netplan/00-installer-config.yaml",
      "sudo rm -f /etc/cloud/cloud.cfg.d/curtin-preserve-sources.cfg",
      "sudo rm -f /etc/cloud/cloud.cfg.d/99-installer.cfg",
      "sudo sync"
    ]
  }

  # SSH Hardening
  provisioner "shell" {
    inline = [
      "echo \"PermitEmptyPasswords no\" | sudo tee -a /etc/ssh/sshd_config",
      "echo \"PermitRootLogin no\" | sudo tee -a /etc/ssh/sshd_config",
      "echo \"Protocol 2\" | sudo tee -a /etc/ssh/sshd_config",
      "echo \"AllowUsers ubuntu\" | sudo tee -a /etc/ssh/sshd_config",
      "echo \"PasswordAuthentication no\" | sudo tee -a /etc/ssh/sshd_config",
      "echo \"ChallengeResponseAuthentication no\" | sudo tee -a /etc/ssh/sshd_config",
      "echo \"AuthenticationMethods publickey\" | sudo tee -a /etc/ssh/sshd_config"
    ]
  }

  # Provisioning the VM Template for Cloud-Init Integration in Proxmox #2
  provisioner "file" {
    source      = "files/99-pve.cfg"
    destination = "/tmp/99-pve.cfg"
  }

  # Provisioning the VM Template for Cloud-Init Integration in Proxmox #3
  provisioner "shell" {
    inline = ["sudo cp /tmp/99-pve.cfg /etc/cloud/cloud.cfg.d/99-pve.cfg"]
  }

}
