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

variable "storage_pool" {
  type    = string
  description = "Proxmox storage pool for VM disk."
}

variable "cloud_init_storage_pool" {
  type    = string
  description = "Proxmox storage pool for cloud-init disk."
}

variable "node" {
  type = string
  decription = "Proxmox node to create the template on."
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

variable "iso_storage_pool" {
  type        = string
  default     = "local"
  description = "Proxmox storage pool for uploaded ISO. Only used with iso_url."
}

variable "disk_format" {
  type    = string
  default = "qcow2"
}
