package main

import (
	"log"

	"github.com/algo7/karpenter-provider-pve/internal/packer"
)

func main() {
	err := packer.RunPacker([]string{"build", "config.pkr.hcl"})
	if err != nil {
		log.Fatalf("Packer build failed: %v", err)
	}
}
