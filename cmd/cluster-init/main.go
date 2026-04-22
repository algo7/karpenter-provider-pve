package main

import (
	"fmt"
	"os"

	"github.com/algo7/karpenter-provider-pve/internal/packer"
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
proxmox_api_url          = "https://pve.example.com:8006/api2/json"
proxmox_api_token_id     = "packer@pve!bootstrap"
proxmox_api_token_secret = "your-token-uuid-here"

# ──────────────────────────────────────────────────────────────────────────────
# Target Proxmox node & storage
# ──────────────────────────────────────────────────────────────────────────────
node                    = "pve-01"
storage_pool            = "local-lvm"
cloud_init_storage_pool = "local-lvm"

# ──────────────────────────────────────────────────────────────────────────────
# Boot ISO — provide EITHER iso_file OR (iso_url + iso_checksum), not both.
# ──────────────────────────────────────────────────────────────────────────────

# Option 1: pre-uploaded ISO on the Proxmox node.
# iso_file = "local:iso/ubuntu-24.04-live-server-amd64.iso"

# Option 2: ISO downloaded by Packer at build time.
iso_url          = "https://releases.ubuntu.com/24.04/ubuntu-24.04-live-server-amd64.iso"
iso_checksum     = "file:https://releases.ubuntu.com/24.04/SHA256SUMS"
iso_storage_pool = "local"

# ──────────────────────────────────────────────────────────────────────────────
# Disk
# ──────────────────────────────────────────────────────────────────────────────
disk_format = "qcow2"
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
	var outPath string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Emit a config file template to stdout or -o <path>. When using -o, please make sure the file has the extension of pkrvars.hcl",
		RunE: func(cmd *cobra.Command, args []string) error {
			if outPath == "" {
				_, err := fmt.Fprint(os.Stdout, configTemplate)
				return err
			}

			if err := os.WriteFile(outPath, []byte(configTemplate), 0o600); err != nil {
				return fmt.Errorf("write config template: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Wrote config template to %s\n", outPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outPath, "output", "o", "", "write template to file instead of stdout")
	return cmd
}

// newBuildCmd returns the `build` subcommand, which runs Packer against
// an embedded template using the user-supplied pkrvars file.
func newBuildCmd() *cobra.Command {
	var (
		configPath   string
		templateName string
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

			return packer.RunPacker(templateName, configPath, []string{"build"})
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "path to pkrvars.hcl config file (required)")
	cmd.Flags().StringVarP(&templateName, "template", "t", "ubuntu", "embedded template to build")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}
