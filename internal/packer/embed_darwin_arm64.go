//go:build darwin

package packer

import "embed"

// Run `make sync-packer` to download the packer binary and the plugin as they are not committed to the repository. The command will place them in the bin/ directory.

//go:embed bin/packer_darwin_arm64 bin/packer-plugin-proxmox_darwin_arm64
var packerBinaries embed.FS

const (
	packerName = "packer_darwin_arm64"
	pluginName = "packer-plugin-proxmox_darwin_arm64"
)

var pluginVersion = "unknown"
