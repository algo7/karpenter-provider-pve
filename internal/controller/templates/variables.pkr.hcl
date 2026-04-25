# Proxmox variables

variable "proxmox_node" {
  type        = string
  description = "Proxmox node on which to run the build."
}

variable "proxmox_api_url" {
  type = string
  description = "Base URL for the Proxmox API (e.g., https://proxmox.example.com:8006/api2/json)."
}

variable "proxmox_api_token_id" {
  type = string
  description = "Proxmox API token ID (e.g., user@realm!tokenid)."
}

variable "proxmox_api_token_secret" {
  type      = string
  sensitive = true
  description = "Proxmox API token secret (the actual token value)."
}

# Storage variables

variable "storage_pool" {
  type    = string
  description = "Storage pool for the resulting template disk."
}

variable "iso_storage_pool" {
  type    = string
  description = "Storage pool used for the installer ISO."
}

variable "disk_format" {
  type    = string
  default = "qcow2"
}

# ISO variables

variable "iso_file" {
  type = string
  description = "Pre-uploaded ISO reference (e.g., local:iso/ubuntu.iso). If set, iso_url and iso_checksum are ignored."
}

variable "iso_checksum" {
  type        = string
  default     = ""
  description = "Checksum of the ISO at iso_url."
}

variable "iso_url" {
  type        = string
  default     = ""
  description = "HTTP(S) URL of the OS installer ISO."
}

# Network variables

variable "network_bridge" {
  type    = string
  description = "Proxmox network bridge to attach VM to."
}

variable "network_vlan_tag" {
  type    = number
  description = "VLAN tag for the network bridge."
}

# Template variables

variable "vm_id" {
  type        = number
  description = "Proxmox VM ID for the builder and resulting template."
}

variable "template_name" {
  type        = string
  description = "Name of the resulting VM template."
}

variable "template_description" {
  type        = string
  default     = ""
  description = "Description annotated on the template."
}

variable "distribution_type" {
  type        = string
  description = "Kubernetes distribution family (rke2, k3s)."
}

variable "distribution_version" {
  type        = string
  description = "Kubernetes distribution version."
}

variable "timezone" {
  type        = string
  default     = "UTC"
  description = "System timezone baked into the image."
}

variable "extra_packages" {
  type        = list(string)
  default     = []
  description = "APT packages installed beyond the controller defaults."
}

variable "ssh_authorized_keys" {
  type        = list(string)
  default     = []
  description = "SSH public keys added to the default user's authorized_keys."
}
