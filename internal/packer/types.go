package packer

// Config represents the variables required by the Packer proxmox-iso plugins
type Config struct {
	ProxmoxAPIURL         string `json:"proxmox_api_url"`
	ProxmoxAPITokenID     string `json:"proxmox_api_token_id"`
	ProxmoxAPITokenSecret string `json:"proxmox_api_token_secret"`
	StoragePool           string `json:"storage_pool"`
	CloudInitStoragePool  string `json:"cloud_init_storage_pool"`
	ISOStoragePool        string `json:"iso_storage_pool"`
	Node                  string `json:"node"`
	ISOFile               string `json:"iso_file"`
	DiskFormat            string `json:"disk_format"` // Default should be "qcow2"
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() Config {
	return Config{
		DiskFormat: "qcow2",
	}
}
