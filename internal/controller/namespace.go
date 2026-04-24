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
	"os"
	"strings"
	"sync"
)

// serviceAccountNamespaceFile is the path Kubernetes mounts the current Pod's
// namespace into when running with a ServiceAccount. This is the canonical
// fallback when POD_NAMESPACE isn't explicitly set via the Downward API.
const serviceAccountNamespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

// podNamespaceEnvVar is the env var populated by the Downward API in the
// controller's Deployment manifest. Preferred over the file because it's
// explicit and survives ServiceAccount token refresh.
const podNamespaceEnvVar = "POD_NAMESPACE"

// defaultNamespace is used only when running outside a cluster (local dev
// via `make run`) and no env var is set. Matches the namespace Kubebuilder's
// scaffolded deployment defaults to.
const defaultNamespace = "karpenter"

var (
	// namespaceOnce guards lazy initialization. The namespace doesn't change
	// at runtime so we resolve it once and cache.
	namespaceOnce   sync.Once
	cachedNamespace string
)

// controllerNamespace returns the namespace the controller is running in.
// Resolution order:
//  1. POD_NAMESPACE environment variable (set via Downward API in-cluster).
//  2. The ServiceAccount namespace file (fallback when the env var isn't set).
//  3. The defaultNamespace constant (for local dev outside the cluster).
//
// Cached after first call because this value never changes during a process's
// lifetime — the Pod's namespace is fixed at scheduling time.
func controllerNamespace() string {
	namespaceOnce.Do(func() {
		// Preferred: explicit env var from Downward API.
		if ns := strings.TrimSpace(os.Getenv(podNamespaceEnvVar)); ns != "" {
			cachedNamespace = ns
			return
		}

		// Fallback: read from the ServiceAccount mount.
		if data, err := os.ReadFile(serviceAccountNamespaceFile); err == nil {
			if ns := strings.TrimSpace(string(data)); ns != "" {
				cachedNamespace = ns
				return
			}
		}

		// Last resort: the hardcoded default for local dev.
		cachedNamespace = defaultNamespace
	})
	return cachedNamespace
}
