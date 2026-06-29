// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

const (
	dockerConfigDirEnv  = "DOCKER_CONFIG"
	xdgRuntimeDirEnv    = "XDG_RUNTIME_DIR"
	dockerConfigFileDir = ".docker"
	dockerConfigFile    = "config.json"
	podmanAuthFile      = "auth.json"
)

// credentialPaths returns an ordered list of credential config file paths
// to check. The order is:
//  1. Docker: $DOCKER_CONFIG/config.json or $HOME/.docker/config.json
//  2. Podman runtime: $XDG_RUNTIME_DIR/containers/auth.json
//  3. Podman config: $HOME/.config/containers/auth.json
func credentialPaths() []string {
	var paths []string

	// 1. Docker config path
	if configDir := os.Getenv(dockerConfigDirEnv); configDir != "" {
		paths = append(paths, filepath.Join(configDir, dockerConfigFile))
	} else if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, dockerConfigFileDir, dockerConfigFile))
	}

	// 2. Podman runtime auth ($XDG_RUNTIME_DIR/containers/auth.json)
	if runtimeDir := os.Getenv(xdgRuntimeDirEnv); runtimeDir != "" {
		paths = append(paths, filepath.Join(runtimeDir, "containers", podmanAuthFile))
	}

	// 3. Podman config auth ($HOME/.config/containers/auth.json)
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".config", "containers", podmanAuthFile))
	}

	return paths
}

// NewCredentialFunc returns an auth.CredentialFunc backed by a credential
// resolution chain that checks Docker and Podman auth locations.
// The resolution order is:
//  1. Docker: $DOCKER_CONFIG/config.json or $HOME/.docker/config.json
//  2. Podman runtime: $XDG_RUNTIME_DIR/containers/auth.json
//  3. Podman config: $HOME/.config/containers/auth.json
//
// Podman paths that do not exist are silently skipped.
func NewCredentialFunc() (auth.CredentialFunc, error) {
	paths := credentialPaths()

	var stores []credentials.Store
	for _, p := range paths {
		store, err := credentials.NewStore(p, credentials.StoreOptions{})
		if err != nil {
			slog.Debug("skipping credential path", "path", p, "error", err)
			continue
		}
		slog.Debug("loaded credential store", "path", p)
		stores = append(stores, store)
	}

	if len(stores) == 0 {
		slog.Debug("no credential stores found, using anonymous auth")
		return func(_ context.Context, _ string) (auth.Credential, error) {
			return auth.EmptyCredential, nil
		}, nil
	}

	combined := credentials.NewStoreWithFallbacks(stores[0], stores[1:]...)
	return credentials.Credential(combined), nil
}
