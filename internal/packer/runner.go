package packer

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

// RunPacker extracts the embedded Packer binary, Proxmox plugin, and the named
// template into a temporary directory, copies the user's pkrvars file into the
// template dir as an auto-loaded var file, and runs Packer. Extra args are
// passed through (e.g. "build", "-force"). The template dir is appended as the
// final positional argument automatically.
func RunPacker(templateName, userVarFile string, args []string) error {
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
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
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

	// Copy user's pkrvars file into the template dir with an auto-load suffix
	dst := filepath.Join(templateDir, "user.auto.pkrvars.hcl")
	if err := copyFile(userVarFile, dst, 0o600); err != nil {
		return fmt.Errorf("copy user config: %w", err)
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

// extractFile reads a single embedded file from packerBinaries and writes it
// to targetPath with executable permissions.
func extractFile(targetPath, embedPath string) error {
	data, err := packerBinaries.ReadFile(embedPath)
	if err != nil {
		return fmt.Errorf("failed to read embedded file %s: %w", embedPath, err)
	}
	return os.WriteFile(targetPath, data, 0o755)
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
			return os.MkdirAll(target, 0o755)
		}

		data, err := packerTemplates.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// copyFile copies src to dst with the given mode.
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
