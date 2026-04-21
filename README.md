# karpenter-provider-pve

An opinionated Karpenter provider for Proxmox VE, targeting self-contained Kubernetes distributions with their own node-join mechanisms.
Supported: RKE2, K3s.
Planned: k0s.
Related: [karpenter-provider-proxmox][kpp] targets distributions using standard kubelet bootstrap (kubeadm, Talos).


[kpp]: https://github.com/sergelogvinov/karpenter-provider-proxmox
