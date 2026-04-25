# Variable Definitions
variable "proxmox_api_url" {
  type = string
}

variable "proxmox_api_token_id" {
  type = string
}

variable "proxmox_api_token_secret" {
  type      = string
  sensitive = true
}

variable "vm_id" {
  type    = string
  default = "9999"
  description = "Proxmox VM ID to use for the template. Must be unique and not conflict with existing VMs. Default to 9999."
}

variable "storage_pool" {
  type    = string
  description = "Proxmox storage pool for VM disk."
}


variable "node" {
  type = string
  description = "Proxmox node to create the template on."
}

variable "iso_storage_pool" {
  type    = string
  description = "Proxmox storage pool to upload ISO to."
}

variable "iso_file" {
  type        = string
  default     = ""
  description = "Proxmox datastore path, e.g. local:iso/ubuntu.iso. Mutually exclusive with iso_url."
}

variable "iso_url" {
  type        = string
  default     = ""
  description = "URL to download ISO from. Mutually exclusive with iso_file."
}

variable "iso_checksum" {
  type        = string
  default     = ""
  description = "Checksum for iso_url. Required when iso_url is set. Format: 'sha256:...' or 'file:<url>'."
}


variable "disk_format" {
  type    = string
  default = "qcow2"
}

variable "network_bridge" {
  type    = string
  description = "Proxmox network bridge to attach VM to."
}

variable "network_vlan_tag" {
  type    = number
  description = "VLAN tag for the network bridge."
}
