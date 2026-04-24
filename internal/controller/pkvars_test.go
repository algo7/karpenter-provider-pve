/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	karpenterv1alpha1 "github.com/algo7/karpenter-provider-pve/api/v1alpha1"
)

func TestRenderPkrvars_URLBased(t *testing.T) {
	in := &pkrvarsInput{
		ProxmoxNode:         "pve-01",
		StoragePool:         "local-lvm",
		ISOStoragePool:      "local-lvm",
		NetworkBridge:       "vmbr0",
		VMID:                9000,
		TemplateName:        "rke2-standard",
		TemplateDescription: "Built by karpenter-provider-pve for PVENodeImage/rke2-standard",
		DistributionType:    "rke2",
		DistributionVersion: "v1.33.4+rke2r1",
		Timezone:            "UTC",
		ExtraPackages:       []string{"qemu-guest-agent", "curl"},
		SSHAuthorizedKeys:   []string{"ssh-ed25519 AAAA test"},
		ISOURL:              "https://example.com/ubuntu.iso",
		ISOChecksum:         "sha256:abc123",
	}

	got := renderPkrvars(in)

	wantLines := []string{
		`proxmox_node = "pve-01"`,
		`storage_pool = "local-lvm"`,
		`iso_storage_pool = "local-lvm"`,
		`network_bridge = "vmbr0"`,
		`vm_id = 9000`,
		`template_name = "rke2-standard"`,
		`distribution_type = "rke2"`,
		`distribution_version = "v1.33.4+rke2r1"`,
		`timezone = "UTC"`,
		`extra_packages = [`,
		`  "qemu-guest-agent",`,
		`  "curl",`,
		`ssh_authorized_keys = [`,
		`  "ssh-ed25519 AAAA test",`,
		`iso_url = "https://example.com/ubuntu.iso"`,
		`iso_checksum = "sha256:abc123"`,
		`iso_file = ""`,
	}

	for _, want := range wantLines {
		if !strings.Contains(got, want) {
			t.Errorf("rendered output missing expected line:\n  want: %q\n  got:\n%s", want, got)
		}
	}
}

func TestRenderPkrvars_ISOFileBased(t *testing.T) {
	in := &pkrvarsInput{
		ProxmoxNode:    "pve-01",
		StoragePool:    "local-lvm",
		ISOStoragePool: "local-lvm",
		NetworkBridge:  "vmbr0",
		VMID:           9000,
		TemplateName:   "from-local-iso",
		ISOFile:        "local:iso/ubuntu-24.04.iso",
	}

	got := renderPkrvars(in)

	if !strings.Contains(got, `iso_file = "local:iso/ubuntu-24.04.iso"`) {
		t.Errorf("missing iso_file line in output:\n%s", got)
	}
	if !strings.Contains(got, `iso_url = ""`) {
		t.Errorf("expected iso_url to be empty string in output:\n%s", got)
	}
}

func TestRenderPkrvars_EmptyLists(t *testing.T) {
	in := &pkrvarsInput{
		VMID:         9000,
		TemplateName: "minimal",
	}

	got := renderPkrvars(in)

	if !strings.Contains(got, `extra_packages = []`) {
		t.Errorf("empty package list should render as []: %s", got)
	}
	if !strings.Contains(got, `ssh_authorized_keys = []`) {
		t.Errorf("empty key list should render as []: %s", got)
	}
}

func TestRenderPkrvars_QuoteEscaping(t *testing.T) {
	in := &pkrvarsInput{
		VMID:              9000,
		TemplateName:      "has\"quotes",
		SSHAuthorizedKeys: []string{`ssh-ed25519 AAA "weird" key`},
	}

	got := renderPkrvars(in)

	// strconv.Quote escapes embedded quotes; verify the output is still
	// parseable HCL by checking the escaped form is present.
	if !strings.Contains(got, `template_name = "has\"quotes"`) {
		t.Errorf("quotes in strings should be escaped:\n%s", got)
	}
	if !strings.Contains(got, `"ssh-ed25519 AAA \"weird\" key"`) {
		t.Errorf("quotes in list items should be escaped:\n%s", got)
	}
}

func TestPkrvarsFromImage_HappyPath(t *testing.T) {
	image := &karpenterv1alpha1.PVENodeImage{
		ObjectMeta: metav1.ObjectMeta{Name: "rke2-standard"},
		Spec: karpenterv1alpha1.PVENodeImageSpec{
			BaseImage: karpenterv1alpha1.BaseImage{
				URL:      "https://example.com/ubuntu.iso",
				Checksum: "sha256:abc123",
			},
			Distribution: karpenterv1alpha1.Distribution{
				Type:    karpenterv1alpha1.DistributionRKE2,
				Version: "v1.33.4+rke2r1",
			},
			Timezone:          "Europe/Zurich",
			Packages:          []string{"qemu-guest-agent"},
			SSHAuthorizedKeys: []string{"ssh-ed25519 AAA user@host"},
			BuildConfig: karpenterv1alpha1.BuildConfig{
				Node:        "pve-01",
				StoragePool: "local-lvm",
				Bridge:      "vmbr0",
			},
		},
	}

	in, err := pkrvarsFromImage(image, 9000, "rke2-standard")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if in.ProxmoxNode != "pve-01" {
		t.Errorf("ProxmoxNode: got %q, want %q", in.ProxmoxNode, "pve-01")
	}
	if in.ISOStoragePool != "local-lvm" {
		t.Errorf("ISOStoragePool should fall back to StoragePool when unset: got %q", in.ISOStoragePool)
	}
	if in.Timezone != "Europe/Zurich" {
		t.Errorf("Timezone: got %q, want %q", in.Timezone, "Europe/Zurich")
	}
	if in.DistributionType != "rke2" {
		t.Errorf("DistributionType: got %q, want %q", in.DistributionType, "rke2")
	}
}

func TestPkrvarsFromImage_MissingRequiredFields(t *testing.T) {
	cases := []struct {
		name    string
		spec    karpenterv1alpha1.PVENodeImageSpec
		wantErr string
	}{
		{
			name: "missing node",
			spec: karpenterv1alpha1.PVENodeImageSpec{
				BuildConfig: karpenterv1alpha1.BuildConfig{
					StoragePool: "local-lvm",
					Bridge:      "vmbr0",
				},
			},
			wantErr: "spec.buildConfig.node is required",
		},
		{
			name: "missing storagePool",
			spec: karpenterv1alpha1.PVENodeImageSpec{
				BuildConfig: karpenterv1alpha1.BuildConfig{
					Node:   "pve-01",
					Bridge: "vmbr0",
				},
			},
			wantErr: "spec.buildConfig.storagePool is required",
		},
		{
			name: "missing bridge",
			spec: karpenterv1alpha1.PVENodeImageSpec{
				BuildConfig: karpenterv1alpha1.BuildConfig{
					Node:        "pve-01",
					StoragePool: "local-lvm",
				},
			},
			wantErr: "spec.buildConfig.bridge is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			image := &karpenterv1alpha1.PVENodeImage{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec:       tc.spec,
			}

			_, err := pkrvarsFromImage(image, 9000, "test")
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error message: got %q, want containing %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestPkrvarsFromImage_ISOStoragePoolFallback(t *testing.T) {
	image := &karpenterv1alpha1.PVENodeImage{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: karpenterv1alpha1.PVENodeImageSpec{
			BuildConfig: karpenterv1alpha1.BuildConfig{
				Node:        "pve-01",
				StoragePool: "fast-storage",
				Bridge:      "vmbr0",
				// ISOStoragePool deliberately unset.
			},
		},
	}

	in, err := pkrvarsFromImage(image, 9000, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if in.ISOStoragePool != "fast-storage" {
		t.Errorf("ISOStoragePool should fall back to StoragePool: got %q", in.ISOStoragePool)
	}
}
