// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"fmt"
	"os"
	"path/filepath"

	"oras.land/oras-go/v2/content/oci"
)

// ResolveDir resolves the cache directory using the following priority:
//  1. explicit flag value (if non-empty)
//  2. $XDG_CACHE_HOME/complypack (if XDG_CACHE_HOME is set and non-empty)
//  3. $HOME/.complypack/cache (fallback)
//
// Returns an error if neither HOME nor XDG_CACHE_HOME is set and no explicit
// value is provided.
func ResolveDir(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "complypack"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot resolve cache directory: HOME is not set and --cache-dir was not provided: %w", err)
	}
	return filepath.Join(home, ".complypack", "cache"), nil
}

// NewOCIStore creates or opens an OCI Image Layout store at the given directory.
// The directory and all parents are created with 0750 permissions if they do
// not already exist.
func NewOCIStore(cacheDir string) (*oci.Store, error) {
	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create cache directory %s: %w", cacheDir, err)
	}
	store, err := oci.New(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open OCI store at %s: %w", cacheDir, err)
	}
	return store, nil
}

// Clean removes all contents of the cache directory. If the directory does not
// exist, Clean returns nil (not an error).
func Clean(cacheDir string) error {
	info, err := os.Stat(cacheDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to access cache directory %s: %w", cacheDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("cache path %s is not a directory", cacheDir)
	}
	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("failed to clean cache directory %s: %w", cacheDir, err)
	}
	return nil
}
