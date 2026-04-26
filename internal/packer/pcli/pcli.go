package pcli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	t "github.com/algo7/karpenter-provider-pve/internal/packer"
)

// RunPacker extracts the embedded Packer binary, Proxmox plugin, and the named
// template into a temporary directory, copies the user's pkrvars file into the
// template dir as an auto-loaded var file, and runs Packer.
func RunPacker(templateName, userVarFile, osVersion string) error {
	fmt.Println("Running Packer with template:", templateName)
	fmt.Println("Plese note that this can take a while as packer is configured to install all the available updates before creating the VM Template")
	templateName = fmt.Sprintf("%s/%s", templateName, osVersion)
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

	// Proxmox plugin — must be namespaced and accompanied by a SHA256SUM file
	// per Packer 1.11+ plugin loading rules.
	pluginRoot := filepath.Join(tmpDir, "plugins")
	pluginNSDir := filepath.Join(pluginRoot, "github.com", "hashicorp", "proxmox")
	if err := os.MkdirAll(pluginNSDir, 0o755); err != nil {
		return fmt.Errorf("failed to create plugin dir: %w", err)
	}

	// Filename: packer-plugin-proxmox_v<version>_x5.0_<os>_<arch>
	pluginFilename := fmt.Sprintf(
		"packer-plugin-proxmox_v%s_x5.0_%s_%s",
		pluginVersion, runtime.GOOS, runtime.GOARCH,
	)
	pluginPath := filepath.Join(pluginNSDir, pluginFilename)
	if err := extractFile(pluginPath, "bin/"+pluginName); err != nil {
		return err
	}

	// Write the companion SHA256SUM file.
	if err := writePluginChecksum(pluginPath); err != nil {
		return fmt.Errorf("write plugin checksum: %w", err)
	}

	// Extract template tree
	// templateDir := filepath.Join(tmpDir, "template")
	// srcDir := filepath.Join("templates", templateName)
	srcDir := "templates"
	if err := extractDir(srcDir, tmpDir); err != nil {
		return fmt.Errorf("failed to extract template %q: %w", templateName, err)
	}

	// Copy user's pkrvars file into the template dir with an auto-load suffix
	dst := filepath.Join(tmpDir, "user.auto.pkrvars.hcl")
	if err := copyFile(userVarFile, dst, 0o600); err != nil {
		return fmt.Errorf("copy user config: %w", err)
	}

	// Append template dir as final positional arg
	cmd := exec.Command(corePath, "build", ".")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PACKER_PLUGIN_PATH=%s", pluginRoot),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// writePluginChecksum computes the sha256 of the plugin binary and writes
// the hex digest to a file named "<pluginPath>_SHA256SUM".
func writePluginChecksum(pluginPath string) error {
	data, err := os.ReadFile(pluginPath)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	hexsum := hex.EncodeToString(sum[:])
	return os.WriteFile(pluginPath+"_SHA256SUM", []byte(hexsum), 0o644)
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

func extractDir(srcDir, destDir string) error {
	return fs.WalkDir(t.TemplateFs, srcDir, func(embeddedPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Split the path to look for our target directories
		segments := strings.Split(embeddedPath, "/")

		// Default behavior: flatten everything to the file name
		// We use path.Base instead of filepath.Base because embedded paths always use forward slashes, even on Windows
		// even tho we don't support Windows, it's good to be consistent with the embedded path format
		relPath := path.Base(embeddedPath)

		// Search for "http" or "files" in the path hierarchy
		// These are directories where the build.pkrvars.hcl expects to find files, so we want to preserve their structure
		for i, seg := range segments {
			if seg == "http" || seg == "files" {
				// If found, preserve the path structure from this folder downward
				relPath = strings.Join(segments[i:], "/")
				break
			}
		}

		// Calculate final destination on the local OS
		target := filepath.Join(destDir, filepath.FromSlash(relPath))

		// Check if this item is part of the "http" or "files" tree we want to preserve
		isPreservedTree := strings.HasPrefix(relPath, "http") || strings.HasPrefix(relPath, "files")

		if d.IsDir() {
			if isPreservedTree {
				// Create the directory since it's part of the preserved structure
				return os.MkdirAll(target, 0o755)
			}
			// Skip directories that we are flattening (e.g., 'ubuntu' or '24.04')
			return nil
		}

		// Read and write the file
		data, err := t.TemplateFs.ReadFile(embeddedPath)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", embeddedPath, err)
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
