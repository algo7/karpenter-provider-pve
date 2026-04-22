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

package cloudprovider

import (
	"context"
	"errors"

	"github.com/awslabs/operatorpkg/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"

	karpenterv1alpha1 "github.com/algo7/karpenter-provider-pve/api/v1alpha1"
)

// CloudProviderName identifies this CloudProvider implementation in logs and metrics.
const CloudProviderName = "proxmox-pve"

// ErrNotImplemented is returned by stubbed methods during v0.1 scaffolding.
var ErrNotImplemented = errors.New("not implemented")

// CloudProvider implements the Karpenter cloudprovider.CloudProvider interface
// for Proxmox Virtual Environment.
type CloudProvider struct {
	kubeClient client.Client
}

// New constructs a CloudProvider instance.
// The controller's main.go wires this up with the manager's client.
func New(kubeClient client.Client) *CloudProvider {
	return &CloudProvider{
		kubeClient: kubeClient,
	}
}

// Compile-time assertion that CloudProvider implements the Karpenter interface.
// If the interface changes, this line will fail to compile and catch the
// mismatch immediately rather than at runtime.
var _ cloudprovider.CloudProvider = (*CloudProvider)(nil)

// Create provisions a new node from the given NodeClaim.
// Karpenter calls this when its scheduler has decided a new node is needed.
func (c *CloudProvider) Create(ctx context.Context, nodeClaim *karpv1.NodeClaim) (*karpv1.NodeClaim, error) {
	return nil, ErrNotImplemented
}

// Delete removes the VM backing the given NodeClaim.
// Must return a NodeClaimNotFoundError if the VM is already gone.
func (c *CloudProvider) Delete(ctx context.Context, nodeClaim *karpv1.NodeClaim) error {
	return ErrNotImplemented
}

// Get retrieves a NodeClaim by its providerID.
// Returns NodeClaimNotFoundError if the underlying VM no longer exists.
func (c *CloudProvider) Get(ctx context.Context, providerID string) (*karpv1.NodeClaim, error) {
	return nil, ErrNotImplemented
}

// List returns all NodeClaims this CloudProvider is managing.
// Used by Karpenter to reconcile cluster state against infrastructure.
func (c *CloudProvider) List(ctx context.Context) ([]*karpv1.NodeClaim, error) {
	return nil, ErrNotImplemented
}

// GetInstanceTypes returns the instance types available for the given NodePool.
// Must return all instance types defined in the referenced NodeClass, even
// those currently unavailable.
func (c *CloudProvider) GetInstanceTypes(ctx context.Context, nodePool *karpv1.NodePool) ([]*cloudprovider.InstanceType, error) {
	return nil, ErrNotImplemented
}

// IsDrifted reports whether a NodeClaim has drifted from the configuration
// it was provisioned under.
func (c *CloudProvider) IsDrifted(ctx context.Context, nodeClaim *karpv1.NodeClaim) (cloudprovider.DriftReason, error) {
	return "", ErrNotImplemented
}

// RepairPolicies returns the list of conditions Karpenter should monitor
// as signals for node repair.
func (c *CloudProvider) RepairPolicies() []cloudprovider.RepairPolicy {
	return nil
}

// Name returns the CloudProvider implementation name.
func (c *CloudProvider) Name() string {
	return CloudProviderName
}

// GetSupportedNodeClasses returns the NodeClass types this CloudProvider recognizes.
// The first element is the default NodeClass.
func (c *CloudProvider) GetSupportedNodeClasses() []status.Object {
	return []status.Object{&karpenterv1alpha1.PVENodeClass{}}
}
