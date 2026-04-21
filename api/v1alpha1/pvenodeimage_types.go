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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DistributionType is the Kubernetes distribution family pre-staged in the image.
// +kubebuilder:validation:Enum=rke2;k3s
type DistributionType string

const (
	// DistributionRKE2 pre-stages the RKE2 distribution binaries and images.
	DistributionRKE2 DistributionType = "rke2"

	// DistributionK3s pre-stages the K3s distribution binaries and images.
	DistributionK3s DistributionType = "k3s"
)

// RebuildPolicyTrigger is the rebuild trigger mode for a PVENodeImage.
// +kubebuilder:validation:Enum=OnSpecChange;Periodic;Manual
type RebuildPolicyTrigger string

const (
	// RebuildOnSpecChange rebuilds whenever the spec hash changes.
	RebuildOnSpecChange RebuildPolicyTrigger = "OnSpecChange"

	// RebuildPeriodic rebuilds on a fixed cadence regardless of spec changes.
	RebuildPeriodic RebuildPolicyTrigger = "Periodic"

	// RebuildManual rebuilds only when the resource is annotated with
	// karpenter.algo7.tools/rebuild=true.
	RebuildManual RebuildPolicyTrigger = "Manual"
)

// PVENodeImageSpec defines the desired state of a Proxmox VM template
// used as the base image for Kubernetes worker nodes.
type PVENodeImageSpec struct {
	// baseImage specifies the source OS image used as the starting point
	// for the Packer build.
	// +required
	BaseImage BaseImage `json:"baseImage"`

	// distribution specifies which Kubernetes distribution to pre-stage
	// in the built template.
	// +required
	Distribution Distribution `json:"distribution"`

	// packages is an optional list of additional APT packages to install
	// during the build, beyond the defaults required for Kubernetes nodes.
	// +optional
	// +listType=set
	Packages []string `json:"packages,omitempty"`

	// sshAuthorizedKeys is an optional list of SSH public keys to inject
	// into the default user's authorized_keys file.
	// +optional
	// +listType=set
	SSHAuthorizedKeys []string `json:"sshAuthorizedKeys,omitempty"`

	// timezone sets the system timezone baked into the image, in IANA format
	// (e.g., "UTC", "Europe/Zurich", "America/New_York").
	// +kubebuilder:default="UTC"
	// +optional
	Timezone string `json:"timezone,omitempty"`

	// buildConfig provides per-image overrides for Proxmox build infrastructure.
	// Empty fields inherit from the controller's configured defaults.
	// +optional
	BuildConfig BuildConfig `json:"buildConfig,omitzero"`

	// rebuildPolicy controls when the controller rebuilds the template.
	// +optional
	RebuildPolicy RebuildPolicy `json:"rebuildPolicy,omitzero"`
}

// BaseImage specifies the source OS image for the Packer build.
// Exactly one of (url + checksum) or isoFile must be set.
// +kubebuilder:validation:XValidation:rule="(has(self.url) && has(self.checksum) && !has(self.isoFile)) || (!has(self.url) && !has(self.checksum) && has(self.isoFile))",message="exactly one of (url+checksum) or isoFile must be set"
type BaseImage struct {
	// url is the HTTP(S) location of an OS installer ISO.
	// When set, checksum must also be set for verification.
	// +optional
	// +kubebuilder:validation:Pattern=`^https?://.+`
	URL string `json:"url,omitempty"`

	// checksum is the checksum of the ISO at url, either as a direct
	// "sha256:<hex>" string or a Packer-style reference like
	// "file:https://example.com/SHA256SUMS".
	// +optional
	Checksum string `json:"checksum,omitempty"`

	// isoFile is the path to a pre-uploaded ISO in Proxmox storage,
	// formatted as "<storage>:iso/<filename>" (e.g., "local:iso/ubuntu-24.04.iso").
	// +optional
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9_-]+:iso/.+\.iso$`
	ISOFile string `json:"isoFile,omitempty"`
}

// Distribution identifies the Kubernetes distribution pre-staged in the image.
type Distribution struct {
	// type is the distribution family.
	// +required
	Type DistributionType `json:"type"`

	// version is the distribution version, including distribution-specific
	// suffixes. Examples: "v1.33.4+rke2r1", "v1.33.4+k3s1".
	// +required
	// +kubebuilder:validation:Pattern=`^v\d+\.\d+\.\d+.*$`
	Version string `json:"version"`
}

// BuildConfig specifies per-image overrides for Proxmox build infrastructure.
// Any field left empty inherits from the controller's configured defaults.
// If neither the per-image value nor the controller default is set for a
// required build parameter, the build fails with a clear status condition.
type BuildConfig struct {
	// node overrides the Proxmox node on which to run the build.
	// +optional
	Node string `json:"node,omitempty"`

	// storagePool overrides the storage pool for the resulting template disk.
	// +optional
	StoragePool string `json:"storagePool,omitempty"`

	// isoStoragePool overrides the storage pool used for the installer ISO
	// and cloud-init drives during the build.
	// +optional
	ISOStoragePool string `json:"isoStoragePool,omitempty"`

	// bridge overrides the network bridge for the builder VM during installation.
	// +optional
	Bridge string `json:"bridge,omitempty"`
}

// RebuildPolicy controls when the controller rebuilds the template.
type RebuildPolicy struct {
	// trigger determines what causes a rebuild.
	// +kubebuilder:default=OnSpecChange
	// +optional
	Trigger RebuildPolicyTrigger `json:"trigger,omitempty"`

	// interval is the rebuild cadence when trigger is Periodic.
	// Ignored for other triggers.
	// +optional
	Interval *metav1.Duration `json:"interval,omitempty"`
}

// PVENodeImageStatus defines the observed state of PVENodeImage.
type PVENodeImageStatus struct {
	// conditions describe the current state of the NodeImage.
	// Standard conditions include Ready, Building, and BuildFailed.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// templateVMID is the Proxmox VMID of the currently available template.
	// Unset until the first successful build completes.
	// +optional
	TemplateVMID *int32 `json:"templateVMID,omitempty"`

	// templateNode is the Proxmox node where the current template lives.
	// +optional
	TemplateNode string `json:"templateNode,omitempty"`

	// observedSpecHash is a hash of the spec fields that affect the built
	// template. A mismatch between this value and the current spec hash
	// indicates a rebuild is needed.
	// +optional
	ObservedSpecHash string `json:"observedSpecHash,omitempty"`

	// lastBuildTime records when the current template was built.
	// +optional
	LastBuildTime *metav1.Time `json:"lastBuildTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=pni,categories=karpenter
// +kubebuilder:printcolumn:name="Distribution",type=string,JSONPath=`.spec.distribution.type`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.distribution.version`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="VMID",type=integer,JSONPath=`.status.templateVMID`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PVENodeImage is the Schema for the pvenodeimages API.
type PVENodeImage struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of PVENodeImage.
	// +required
	Spec PVENodeImageSpec `json:"spec"`

	// status defines the observed state of PVENodeImage.
	// +optional
	Status PVENodeImageStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PVENodeImageList contains a list of PVENodeImage.
type PVENodeImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PVENodeImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PVENodeImage{}, &PVENodeImageList{})
}
