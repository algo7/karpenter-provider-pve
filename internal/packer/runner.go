package packer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// RunPacker extracts the embedded Packer binary and Proxmox plugin to a temporary directory and executes Packer with the provided arguments.
func RunPacker(args []string) error {
	// Create a temporary directory to hold the extracted Packer binary and plugin
	tmpDir, err := os.MkdirTemp("", "pve-packer-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Extract Packer Core
	corePath := filepath.Join(tmpDir, "packer")
	if err := extractFile(corePath, "bin/"+packerName); err != nil {
		return err
	}

	// Extract Proxmox Plugin
	// Packer looks for plugins in a specific structure or via PACKER_PLUGIN_PATH
	pluginDir := filepath.Join(tmpDir, "plugins")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return err
	}

	pluginPath := filepath.Join(pluginDir, "packer-plugin-proxmox")
	if err := extractFile(pluginPath, "bin/"+pluginName); err != nil {
		return err
	}

	// Execute
	cmd := exec.Command(corePath, args...)

	// Set Env so Packer finds the bundled plugin and doesn't try to download it
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PACKER_PLUGIN_PATH=%s", pluginDir),
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// extractFile reads the embedded file from the given embedPath and writes it to the targetPath with executable permissions.
func extractFile(targetPath, embedPath string) error {
	data, err := packerBinaries.ReadFile(embedPath)
	if err != nil {
		return fmt.Errorf("failed to read embedded file %s: %w", embedPath, err)
	}
	return os.WriteFile(targetPath, data, 0755)
}
