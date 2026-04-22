package packer

// Config represents the variables required by the Packer proxmox-iso plugins
type Config struct {
	ProxmoxAPIURL         string
	ProxmoxAPITokenID     string
	ProxmoxAPITokenSecret string
	StoragePool           string
	CloudInitStoragePool  string
	Node                  string
	ISOFile               string
	ISOURL                string
	ISOChecksum           string
	ISOStoragePool        string
	DiskFormat            string // Default should be "qcow2"
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() Config {
	return Config{
		DiskFormat: "qcow2",
	}
}
