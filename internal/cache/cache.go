// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"fmt"
	"os"
	"path/filepath"

	"oras.land/oras-go/v2/content/oci"
)

// CacheDirHelp is the user-facing description for --cache-dir flag help text.
const CacheDirHelp = "Cache directory (default: $XDG_CACHE_HOME/complypack or $HOME/.cache/complypack)"

// ResolveDir resolves the cache directory using the following priority:
//  1. explicit flag value (if non-empty)
//  2. $XDG_CACHE_HOME/complypack (if XDG_CACHE_HOME is set and non-empty)
//  3. $HOME/.cache/complypack (per XDG Base Directory Specification fallback)
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
	return filepath.Join(home, ".cache", "complypack"), nil
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

// Clean removes all contents of the cache directory while preserving the
// directory itself. This avoids invalidating the directory inode, which could
// cause issues for other processes that hold a reference to it.
// If the directory does not exist, Clean returns nil (not an error).
func Clean(cacheDir string) error {
	entries, err := os.ReadDir(cacheDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read cache directory %s: %w", cacheDir, err)
	}

	for _, entry := range entries {
		path := filepath.Join(cacheDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}
	return nil
}
