package packer

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RunPacker extracts the embedded Packer binary, Proxmox plugin, and the named
// template into a temporary directory, writes cfg as an auto-loaded var file,
// then runs Packer against the extracted template dir. Extra args are passed
// through (e.g. "build", "-force"). The template dir is appended as the final
// positional argument automatically.
func RunPacker(templateName string, cfg Config, args []string) error {
	tmpDir, err := os.MkdirTemp("", "pve-packer-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Packer core binary
	corePath := filepath.Join(tmpDir, "packer")
	if err := extractFile(corePath, "bin/"+packerName); err != nil {
		return err
	}

	// Proxmox plugin
	pluginDir := filepath.Join(tmpDir, "plugins")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin dir: %w", err)
	}
	pluginPath := filepath.Join(pluginDir, "packer-plugin-proxmox")
	if err := extractFile(pluginPath, "bin/"+pluginName); err != nil {
		return err
	}

	// Extract template tree
	templateDir := filepath.Join(tmpDir, "template")
	srcDir := filepath.Join("templates", templateName)
	if err := extractDir(srcDir, templateDir); err != nil {
		return fmt.Errorf("failed to extract template %q: %w", templateName, err)
	}

	// Write config as auto-loaded HCL var file
	if err := writeVarFile(templateDir, cfg); err != nil {
		return fmt.Errorf("failed to write var file: %w", err)
	}

	// Append template dir as final positional arg
	cmdArgs := append(append([]string{}, args...), templateDir)

	cmd := exec.Command(corePath, cmdArgs...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PACKER_PLUGIN_PATH=%s", pluginDir),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// extractFile reads a single embedded file from packerBinaries and writes it to
// targetPath with executable permissions. Intended for binaries.
func extractFile(targetPath, embedPath string) error {
	data, err := packerBinaries.ReadFile(embedPath)
	if err != nil {
		return fmt.Errorf("failed to read embedded file %s: %w", embedPath, err)
	}
	return os.WriteFile(targetPath, data, 0755)
}

// extractDir walks srcDir inside packerTemplates and mirrors its contents to
// destDir on disk. Directories are created with 0755, files written with 0644.
func extractDir(srcDir, destDir string) error {
	return fs.WalkDir(packerTemplates, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		data, err := packerTemplates.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}
		return os.WriteFile(target, data, 0644)
	})
}

// writeVarFile writes cfg as a Packer auto-loaded HCL var file inside
// templateDir. Empty string fields are skipped so HCL variable defaults take
// effect. The file has 0600 perms since it contains the API token secret.
func writeVarFile(templateDir string, cfg Config) error {
	pairs := []struct{ key, val string }{
		{"proxmox_api_url", cfg.ProxmoxAPIURL},
		{"proxmox_api_token_id", cfg.ProxmoxAPITokenID},
		{"proxmox_api_token_secret", cfg.ProxmoxAPITokenSecret},
		{"storage_pool", cfg.StoragePool},
		{"cloud_init_storage_pool", cfg.CloudInitStoragePool},
		{"node", cfg.Node},
		{"iso_file", cfg.ISOFile},
		{"iso_url", cfg.ISOURL},
		{"iso_checksum", cfg.ISOChecksum},
		{"iso_storage_pool", cfg.ISOStoragePool},
		{"disk_format", cfg.DiskFormat},
	}

	var b strings.Builder
	for _, p := range pairs {
		if p.val == "" {
			continue
		}
		fmt.Fprintf(&b, "%s = %q\n", p.key, p.val)
	}

	varFile := filepath.Join(templateDir, "config.auto.pkrvars.hcl")
	return os.WriteFile(varFile, []byte(b.String()), 0600)
}
