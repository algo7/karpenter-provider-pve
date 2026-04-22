source "proxmox-iso" "default" {

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
    unmount = true
  }


  node         = var.node
  vm_id        = "9999"
  ssh_username = "ubuntu"
  ssh_password = "ubuntu"
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

  # Cloud-init configuration
  cloud_init              = true
  cloud_init_storage_pool = var.cloud_init_storage_pool
    cloud_init_disk_type = "scsi"
  # http_directory          = "http"
  # http_port_min           = 12234
  # http_port_max           = 12234
  additional_iso_files {
    cd_files = [
      "./http/meta-data",
      "./http/user-data"
    ]
    type = "scsi"
    cd_label         = "cidata"
    iso_storage_pool = var.iso_storage_pool
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
