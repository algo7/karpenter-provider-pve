//go:build linux

package pcli

import "embed"

// Run `make sync-packer` to download the packer binary and the plugin as they are not committed to the repository. The command will place them in the bin/ directory.

//go:embed bin/packer_linux_amd64 bin/packer-plugin-proxmox_linux_amd64
var packerBinaries embed.FS

const (
	packerName = "packer_linux_amd64"
	pluginName = "packer-plugin-proxmox_linux_amd64"
)

var pluginVersion = "unknown"
