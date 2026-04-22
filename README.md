# karpenter-provider-pve

An opinionated Karpenter provider for Proxmox VE, targeting self-contained Kubernetes distributions with their own node-join mechanisms.
Supported: RKE2, K3s.
Planned: k0s, Talos.
Related: [karpenter-provider-proxmox][kpp] targets standard kubelet bootstrap.

[kpp]: https://github.com/sergelogvinov/karpenter-provider-proxmox

# Prerequisites

- Proxmox VE 9.1+ with API access
- Supported Kubernetes distribution (RKE2, K3s, etc.) with master/control-plane nodes already set up. Karpenter will manage only worker nodes.

# cluster-init tools

To simplify cluster initialization, you can use the `cluster-init` tools provided in this repository. These tools automate the process of setting up a supported Kubernetes distribution on Proxmox VE, including the creation of the VM template and the initial cluster configuration. The tool will not install Karpenter itself, but it will prepare the environment for you to easily deploy Karpenter and start managing your worker nodes.

The tool uses the packer (embedded in the `cluster-init` binary and extracted to a temporary directory) to create a VM template based on the specified Kubernetes distribution. It then configures the cluster according to the provided parameters, such as the number of control-plane nodes and the desired Kubernetes version.

> [!CAUTION]
> This project is still in early development. The controller does not work yet, and the cluster-init tools only supports Ubuntu 22.04x with VM Template creation only.
