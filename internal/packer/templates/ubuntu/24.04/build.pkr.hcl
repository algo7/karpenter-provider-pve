source "proxmox-iso" "ubuntu" {

  # Proxmox Connection Settings
  proxmox_url              = var.proxmox_api_url
  username                 = var.proxmox_api_token_id
  token                    = var.proxmox_api_token_secret
  insecure_skip_tls_verify = true

  # VM ISO Settings
  boot_iso {
    iso_url                  = var.iso_url
    iso_checksum             = var.iso_checksum
    iso_file     = var.iso_file
    iso_storage_pool = try(var.iso_storage_pool, var.storage_pool)
    iso_download_pve = true
    unmount = true
  }


  node         = var.proxmox_node
  vm_id        = var.vm_id
  # Build-time SSH settings for Packer to connect to the VM and run provisioners. These should align with the user-data configuration to ensure Packer can connect successfully during the build
  ssh_username = "ubuntu" # align with the user-data in files/user-data
  ssh_password = "ubuntu" # align with the user-data in files/user-data
  ssh_timeout  = "20m"

  # VM configuration
  ## Hardware
  ## See: https://docs.rke2.io/install/requirements
  memory   = 4096
  cores    = 2
  sockets  = 1
  cpu_type = "x86-64-v2-AES"
  disks {
    type         = "scsi"
    storage_pool = var.storage_pool
    disk_size    = "30G"
    ssd          = true
    format       =  var.disk_format
  }

  ## OS and BIOS
  os      = "l26"
  bios    = "ovmf"
  machine = "pc"
  efi_config {
    efi_storage_pool  = var.storage_pool
    pre_enrolled_keys = true
    efi_type          = "4m"
  }

  ## Network
  network_adapters {
    model    = "virtio"
    bridge   = var.network_bridge
    vlan_tag = var.network_vlan_tag
    firewall = true
  }

  ## Others
  qemu_agent           = true
  scsi_controller      = "virtio-scsi-single"
  onboot               = true
  template_name        = "ubuntu-24.04-lts-server-standard"
  template_description = "Ubuntu 24.04 LTS Standard Server with 2C4T and 8GB RAM"


  additional_iso_files {
    cd_files = [
      "./http/meta-data",
      "./http/user-data"
    ]
    type = "scsi"
    cd_label         = "cidata"
    iso_storage_pool = try(var.iso_storage_pool, var.storage_pool)
    unmount          = true
  }


  ## Boot options
  boot_wait = "10s"
  boot_command = [
    "<esc><wait>",
    "e<wait>",
    "<down><down><down><end>",
    "<bs><bs><bs><bs><wait>",
    " autoinstall quiet ds=nocloud",
    "<f10><wait>",
    "<wait1m>",
    "yes<enter>"
  ]
}


build {
  name    = "ubuntu"
  sources = ["source.proxmox-iso.ubuntu"]

  # Wait for cloud-init to finish before any customization. Using the
  # boot-finished marker rather than `cloud-init status --wait` works
  # across cloud-init versions
  provisioner "shell" {
    inline = [
      "while [ ! -f /var/lib/cloud/instance/boot-finished ]; do echo 'Waiting for cloud-init...'; sleep 1; done",
    ]
  }

  # Set the system timezone from the CR spec.
  provisioner "shell" {
    inline = [
      "sudo timedatectl set-timezone ${var.timezone}",
    ]
  }

  # Install extra APT packages declared in the CR spec. The `for` expression
  # produces one `apt-get install` line per package; concat prepends the
  # update step. Empty package list becomes just the update
  provisioner "shell" {
    inline = concat(
      ["sudo apt-get update"],
      [for pkg in var.extra_packages : "sudo apt-get install -y ${pkg}"],
    )
  }

  # Inject SSH authorized keys for the ubuntu user. Uses concat so an empty
  # key list skips the loop but still ensures ~/.ssh exists with safe perms
  provisioner "shell" {
    inline = concat(
      [
        "mkdir -p /home/ubuntu/.ssh",
        "chmod 700 /home/ubuntu/.ssh",
      ],
      [for key in var.ssh_authorized_keys : "echo '${key}' | sudo tee -a /home/ubuntu/.ssh/authorized_keys"],
      [
        "sudo chmod 600 /home/ubuntu/.ssh/authorized_keys",
        "sudo chown -R ubuntu:ubuntu /home/ubuntu/.ssh",
      ],
    )
  }

  # SSH hardening. These settings are baked into the template so every
  # cloned VM starts with secure defaults
  provisioner "shell" {
    inline = [
      "echo 'PermitEmptyPasswords no' | sudo tee -a /etc/ssh/sshd_config",
      "echo 'PermitRootLogin no' | sudo tee -a /etc/ssh/sshd_config",
      "echo 'Protocol 2' | sudo tee -a /etc/ssh/sshd_config",
      "echo 'AllowUsers ubuntu' | sudo tee -a /etc/ssh/sshd_config",
      "echo 'PasswordAuthentication no' | sudo tee -a /etc/ssh/sshd_config",
      "echo 'ChallengeResponseAuthentication no' | sudo tee -a /etc/ssh/sshd_config",
      "echo 'AuthenticationMethods publickey' | sudo tee -a /etc/ssh/sshd_config",
    ]
  }

  # Configure cloud-init for Proxmox's cidata delivery mechanism. Required
  # for cloned VMs to correctly pick up their user-data at first boot
  provisioner "file" {
    source      = "files/99-pve.cfg"
    destination = "/tmp/99-pve.cfg"
  }
  provisioner "shell" {
    inline = [
      "sudo cp /tmp/99-pve.cfg /etc/cloud/cloud.cfg.d/99-pve.cfg",
    ]
  }

  # Pre-stage the Kubernetes distribution binaries.
  # TODO: add distribution-specific install logic (RKE2 / K3s) here
  provisioner "shell" {
    environment_vars = [
      "DISTRIBUTION_TYPE=${var.distribution_type}",
      "DISTRIBUTION_VERSION=${var.distribution_version}",
    ]
    inline = [
      "echo \"Pre-staging $DISTRIBUTION_TYPE $DISTRIBUTION_VERSION (placeholder)\"",
    ]
  }

  # Final cleanup before converting to template. Must be last — any
  # provisioner after this would run on a half-wiped system and leak state
  # into the template
  provisioner "shell" {
    inline = [
      # Remove SSH host keys so every clone regenerates unique ones
      "sudo rm -f /etc/ssh/ssh_host_*",
      # Clear machine-id so cloud-init treats each clone as a fresh instance
      "sudo truncate -s 0 /etc/machine-id",
      # Remove subiquity's installer-time netplan config
      "sudo rm -f /etc/cloud/cloud.cfg.d/subiquity-disable-cloudinit-networking.cfg",
      "sudo rm -f /etc/netplan/00-installer-config.yaml",
      # Reset cloud-init state (empties /var/lib/cloud) so first-boot provisioning runs on clones
      "sudo cloud-init clean",
      # APT hygiene.
      "sudo apt-get -y autoremove --purge",
      "sudo apt-get -y clean",
      "sudo apt-get -y autoclean",
      "sudo rm -rf /var/lib/apt/lists/*",
      # Clear root's authorized_keys that Packer added during the build
      "sudo rm -f /root/.ssh/authorized_keys",
      # Clear bash history for ubuntu and root users to avoid leaking build-time
      "sudo rm -f /home/ubuntu/.bash_history /root/.bash_history",
      "history -c || true", # || true is to handle case when shell history is disabled which returns non-zero exit code and causes packer build to fail
      "unset HISTFILE", # Unset HISTFILE to prevent any further history from being written after the build finishes
      "sudo sync",
    ]
  }

  # Write a build manifest the controller parses to extract the resulting
  # template's VMID
  post-processor "manifest" {
    output     = "/workspace/manifest.json"
    strip_path = true
  }
}
