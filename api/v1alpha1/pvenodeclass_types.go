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
	"github.com/awslabs/operatorpkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition types reported on PVENodeClass.Status.Conditions.
const (
	// ConditionTypeTemplateResolved indicates spec.nodeImageRef resolves
	// to an existing PVENodeImage in the cluster.
	ConditionTypeTemplateResolved = "TemplateResolved"

	// ConditionTypeNodeImageReady indicates the referenced PVENodeImage
	// has successfully built its template and reports Ready=True.
	ConditionTypeNodeImageReady = "NodeImageReady"
)

// PVENodeClassSpec defines the desired state of a PVENodeClass, which
// describes how to clone and configure Proxmox VMs for Kubernetes worker
// nodes provisioned by Karpenter.
type PVENodeClassSpec struct {
	// nodeImageRef references the PVENodeImage used as the base template
	// for nodes provisioned under this NodeClass. The referenced image
	// must be cluster-scoped and reach Ready=True before nodes can be
	// provisioned.
	// +required
	NodeImageRef NodeImageReference `json:"nodeImageRef"`

	// instanceTypes defines the set of VM shapes exposed to Karpenter's
	// scheduler for this NodeClass. Each entry becomes a selectable
	// instance type via the node.kubernetes.io/instance-type label.
	// +required
	// +kubebuilder:validation:MinItems=1
	// +listType=map
	// +listMapKey=name
	InstanceTypes []InstanceType `json:"instanceTypes"`

	// placementTargets lists the Proxmox hypervisor nodes (hosts) available
	// for VM placement within the configured Proxmox cluster, along with the
	// Karpenter topology.kubernetes.io/zone label each represents. At least
	// one entry is required; single-host Proxmox (non-cluster) setups use a single entry.
	// Multi-cluster Proxmox deployments (a Kubernetes cluster that spans across multiple Proxmox clusters) are not yet supported.
	// +required
	// +kubebuilder:validation:MinItems=1
	// +listType=map
	// +listMapKey=node
	PlacementTargets []PlacementTarget `json:"placementTargets"`

	// storagePool is the Proxmox storage pool that backs cloned VM disks.
	// +required
	StoragePool string `json:"storagePool"`

	// network configures networking for cloned VMs.
	// +required
	Network NetworkConfig `json:"network"`

	// tags are additional Proxmox tags applied to cloned VMs. The
	// controller always adds its own management tag for filtering during
	// List operations; user-provided tags extend that set.
	// +optional
	// +listType=set
	Tags []string `json:"tags,omitempty"`
}

// NodeImageReference identifies a cluster-scoped PVENodeImage by name.
type NodeImageReference struct {
	// name of the referenced PVENodeImage.
	// +required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// InstanceType defines a named VM shape that maps to a Karpenter instance
// type. Karpenter's scheduler uses these to simulate node-addition decisions
// against pending pods.
type InstanceType struct {
	// name is the instance type identifier (e.g., "small", "medium", "gpu").
	// Surfaced to Karpenter as node.kubernetes.io/instance-type.
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[a-z0-9][a-z0-9-]*$`
	Name string `json:"name"`

	// cpu is the number of vCPUs for this instance type.
	// +required
	// +kubebuilder:validation:Minimum=1
	CPU int32 `json:"cpu"`

	// memoryMiB is the amount of RAM in MiB for this instance type.
	// +required
	// +kubebuilder:validation:Minimum=512
	MemoryMiB int32 `json:"memoryMiB"`

	// diskGiB is the boot disk size in GiB for this instance type.
	// Must be greater than or equal to the disk size baked into the
	// referenced PVENodeImage, or Proxmox will reject the clone.
	// +required
	// +kubebuilder:validation:Minimum=10
	DiskGiB int32 `json:"diskGiB"`
}

// PlacementTarget maps a Proxmox node to a Karpenter topology zone.
type PlacementTarget struct {
	// node is the Proxmox node name (e.g., "pve-01") where VMs may be placed.
	// +required
	// +kubebuilder:validation:MinLength=1
	Node string `json:"node"`

	// zone is the Karpenter topology.kubernetes.io/zone label value this
	// Proxmox node represents. If empty, the node name is used as the zone.
	// +optional
	Zone string `json:"zone,omitempty"`
}

// NetworkConfig specifies network settings applied to cloned VMs.
type NetworkConfig struct {
	// bridge is the Proxmox network bridge attached to cloned VMs.
	// +required
	// +kubebuilder:validation:MinLength=1
	Bridge string `json:"bridge"`

	// vlanTag is an optional VLAN ID for the VM network interface.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=4094
	VLANTag *int32 `json:"vlanTag,omitempty"`

	// firewall enables the Proxmox firewall on the VM network interface.
	// +kubebuilder:default=false
	// +optional
	Firewall bool `json:"firewall,omitempty"`
}

// PVENodeClassStatus defines the observed state of a PVENodeClass.
type PVENodeClassStatus struct {
	// conditions describe the current state of the NodeClass. The root
	// Ready condition aggregates from TemplateResolved and NodeImageReady.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=pnc,categories=karpenter
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.nodeImageRef.name`
// +kubebuilder:printcolumn:name="InstanceTypes",type=integer,JSONPath=`.spec.instanceTypes.length()`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PVENodeClass is the Schema for the pvenodeclasses API.
type PVENodeClass struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of PVENodeClass.
	// +required
	Spec PVENodeClassSpec `json:"spec"`

	// status defines the observed state of PVENodeClass.
	// +optional
	Status PVENodeClassStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PVENodeClassList contains a list of PVENodeClass.
type PVENodeClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PVENodeClass `json:"items"`
}

// GetConditions returns the status conditions as operatorpkg's Condition type,
// satisfying the status.Object interface required by the Karpenter
// CloudProvider's GetSupportedNodeClasses method.
func (in *PVENodeClass) GetConditions() []status.Condition {
	out := make([]status.Condition, len(in.Status.Conditions))
	for i, c := range in.Status.Conditions {
		out[i] = status.Condition(c)
	}
	return out
}

// SetConditions replaces the status conditions, converting from operatorpkg's
// Condition type to metav1.Condition for persistence in status.
// Required by the status.Object interface.
func (in *PVENodeClass) SetConditions(conditions []status.Condition) {
	out := make([]metav1.Condition, len(conditions))
	for i, c := range conditions {
		out[i] = metav1.Condition(c)
	}
	in.Status.Conditions = out
}

// StatusConditions returns a ConditionSet that aggregates the listed
// subconditions into the root Ready condition.
// Required by the status.Object interface.
func (in *PVENodeClass) StatusConditions() status.ConditionSet {
	return status.NewReadyConditions(
		ConditionTypeTemplateResolved,
		ConditionTypeNodeImageReady,
	).For(in)
}

func init() {
	SchemeBuilder.Register(&PVENodeClass{}, &PVENodeClassList{})
}
