// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"os"
	"path/filepath"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

const (
	xdgRuntimeDirEnv = "XDG_RUNTIME_DIR"
	podmanAuthFile   = "auth.json"
)

// podmanPaths returns Podman credential file paths to check as fallbacks.
// The order is:
//  1. Podman runtime: $XDG_RUNTIME_DIR/containers/auth.json
//  2. Podman config: $HOME/.config/containers/auth.json
func podmanPaths() []string {
	var paths []string

	// 1. Podman runtime auth ($XDG_RUNTIME_DIR/containers/auth.json)
	if runtimeDir := os.Getenv(xdgRuntimeDirEnv); runtimeDir != "" {
		paths = append(paths, filepath.Join(runtimeDir, "containers", podmanAuthFile))
	}

	// 2. Podman config auth ($HOME/.config/containers/auth.json)
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".config", "containers", podmanAuthFile))
	}

	return paths
}

// NewCredentialFunc returns an auth.CredentialFunc backed by a credential
// resolution chain that checks Docker and Podman auth locations.
// The resolution order is:
//  1. Docker: via NewStoreFromDocker (handles $DOCKER_CONFIG and $HOME/.docker/config.json)
//  2. Podman runtime: $XDG_RUNTIME_DIR/containers/auth.json
//  3. Podman config: $HOME/.config/containers/auth.json
//
// Podman paths that do not exist are silently skipped.
func NewCredentialFunc() (auth.CredentialFunc, error) {
	dockerStore, err := credentials.NewStoreFromDocker(credentials.StoreOptions{})
	if err != nil {
		return nil, err
	}

	var podmanStores []credentials.Store
	for _, p := range podmanPaths() {
		store, err := credentials.NewStore(p, credentials.StoreOptions{})
		if err != nil {
			continue
		}
		podmanStores = append(podmanStores, store)
	}

	combined := credentials.NewStoreWithFallbacks(dockerStore, podmanStores...)
	return credentials.Credential(combined), nil
}
