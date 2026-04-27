package main

import (
	"fmt"
	"os"

	"github.com/algo7/karpenter-provider-pve/internal/packer/pcli"
	"github.com/spf13/cobra"
)

// configTemplate is the raw pkrvars.hcl template emitted by `cluster-init init`.
// Users fill this in and pass the resulting file to `cluster-init build -c <file>`.
const configTemplate = `# cluster-init configuration
# Fill in the values below, then run:
#   cluster-init build -c <path-to-this-file>

# ──────────────────────────────────────────────────────────────────────────────
# Proxmox connection
# ──────────────────────────────────────────────────────────────────────────────
proxmox_node             = "pve"
proxmox_api_url          = "https://pve.example.com:8006/api2/json"
proxmox_api_token_id     = "packer@pve!bootstrap"
proxmox_api_token_secret = "your-token-uuid-here"

# ──────────────────────────────────────────────────────────────────────────────
# Target Proxmox node & storage
# ──────────────────────────────────────────────────────────────────────────────
storage_pool            = "local"

# Storage pool to store the downloaded ISO. Defaults to storage_pool if not set.
# iso_storage_pool      = "local"

# ──────────────────────────────────────────────────────────────────────────────
# Boot ISO — provide EITHER iso_file OR (iso_url + iso_checksum), not both.
# ──────────────────────────────────────────────────────────────────────────────

# Option 1: path to pre-uploaded ISO on the Proxmox node.
iso_file = "local:iso/ubuntu-24.04-live-server-amd64.iso"

# Option 2: ISO downloaded by Packer at build time.
iso_url          = "https://releases.ubuntu.com/24.04/ubuntu-24.04-live-server-amd64.iso"

## iso_checksum can be a raw SHA256 hash or
## a URL with "file:" prefix pointing to a file containing the hash.
## Defaults to none.
iso_checksum     = "file:https://releases.ubuntu.com/24.04/SHA256SUMS"



# ──────────────────────────────────────────────────────────────────────────────
# Disk
# ──────────────────────────────────────────────────────────────────────────────
disk_format = "qcow2"

# ──────────────────────────────────────────────────────────────────────────────
# Network - for VM network access during build. This should be a bridge with internet access,
# but it doesn't have to be the same one used by the final cluster VMs.
# ──────────────────────────────────────────────────────────────────────────────
network_bridge = "vmbr0"

# Optional VLAN tag for the network interface. Defaults to untagged if not set.
# network_vlan_tag = "20"

# ──────────────────────────────────────────────────────────────────────────────
# VM Settings
# ──────────────────────────────────────────────────────────────────────────────
vm_id = 9000

# IANA timezone
timezone = "UTC"

# ──────────────────────────────────────────────────────────────────────────────
# Template settings
# ──────────────────────────────────────────────────────────────────────────────
template_name = "ubuntu-2404-live-server"
template_description = "Ubuntu 24.04 LTS live server template for karpenter-provider-pve"
`

func main() {
	root := &cobra.Command{
		Use:   "cluster-init",
		Short: "Bootstrap Proxmox VM templates for karpenter-provider-pve",
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(newInitCmd(), newBuildCmd())

	if err := root.Execute(); err != nil {
		// Cobra already prints the error; exit non-zero so scripts see it.
		os.Exit(1)
	}
}

// newInitCmd returns the `init` subcommand, which emits a config template
// either to stdout or to a file via -o.
func newInitCmd() *cobra.Command {
	var outPutName string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Emit a config file template to stdout or -o <file_name>. When using -o, `.pkrvars.hcl` is automatically appended",
		RunE: func(cmd *cobra.Command, args []string) error {
			if outPutName == "" {
				_, err := fmt.Fprint(os.Stdout, configTemplate)
				return err
			}
			// Automatically append .pkrvars.hcl
			outPutNameWithExt := fmt.Sprintf("%s.pkrvars.hcl", outPutName)
			if err := os.WriteFile(outPutNameWithExt, []byte(configTemplate), 0o600); err != nil {
				return fmt.Errorf("write config template: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Wrote config template to %s\n", outPutNameWithExt)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outPutName, "output", "o", "", "write template to file instead of stdout (optional, .pkrvars.hcl automatically appended)")
	return cmd
}

// newBuildCmd returns the `build` subcommand, which runs Packer against
// an embedded template using the user-supplied pkrvars file.
func newBuildCmd() *cobra.Command {
	var (
		configPath   string
		templateName string
		osVersion    string
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a Proxmox VM template from an embedded Packer template",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Fail fast if the config file doesn't exist, rather than letting
			// Packer report it after all the extraction work.
			if _, err := os.Stat(configPath); err != nil {
				return fmt.Errorf("config file: %w", err)
			}

			return pcli.RunPacker(templateName, configPath, osVersion)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "path to pkrvars.hcl config file (required)")
	cmd.Flags().StringVarP(&templateName, "template", "t", "ubuntu", "embedded template to build")
	cmd.Flags().StringVarP(&osVersion, "os-version", "v", "", "OS version to build (optional, e.g. '24.04'; only needed if template supports multiple OS versions)")
	_ = cmd.MarkFlagRequired("config")
	_ = cmd.MarkFlagRequired("template")
	_ = cmd.MarkFlagRequired("os-version")

	return cmd
}
