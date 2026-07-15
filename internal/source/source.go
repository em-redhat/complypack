// SPDX-License-Identifier: Apache-2.0

// Package source resolves Gemara artifact sources (file paths or OCI references)
// and loads them into classified ArtifactSets.
package source

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/complytime/complypack/internal/cache"
	"github.com/complytime/complypack/internal/registry"
	"github.com/complytime/complypack/internal/requirement"
	"github.com/gemaraproj/go-gemara/bundle"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/content/oci"
)

// ociStoreCache caches OCI store instances by directory path to avoid
// re-walking the OCI layout on every loadBundleArtifacts call.
var (
	ociStoreMu    sync.Mutex
	ociStoreCache = make(map[string]*oci.Store)
)

// getOrCreateOCIStore returns a cached OCI store for the given directory,
// creating it if needed.
func getOrCreateOCIStore(cacheDir string) (*oci.Store, error) {
	ociStoreMu.Lock()
	defer ociStoreMu.Unlock()

	if store, ok := ociStoreCache[cacheDir]; ok {
		return store, nil
	}

	store, err := cache.NewOCIStore(cacheDir)
	if err != nil {
		return nil, err
	}
	ociStoreCache[cacheDir] = store
	return store, nil
}

// LoadArtifacts loads and classifies Gemara artifacts from either a file path or OCI reference.
// When cacheDir is non-empty, OCI artifacts are cached on disk for subsequent invocations.
func LoadArtifacts(ctx context.Context, source string, plainHTTP bool, cacheDir string) (*requirement.ArtifactSet, error) {
	if strings.HasPrefix(source, "file://") {
		path := strings.TrimPrefix(source, "file://")
		return loadFileArtifacts(ctx, path)
	}

	if strings.HasPrefix(source, "oci://") {
		ref := strings.TrimPrefix(source, "oci://")
		return loadBundleArtifacts(ctx, ref, plainHTTP, cacheDir)
	}

	if IsOCIReference(source) {
		return loadBundleArtifacts(ctx, source, plainHTTP, cacheDir)
	}

	return loadFileArtifacts(ctx, source)
}

func loadFileArtifacts(_ context.Context, path string) (*requirement.ArtifactSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	result, err := requirement.Classify(data)
	if err != nil {
		return nil, fmt.Errorf("failed to classify artifact: %w", err)
	}

	return result, nil
}

func loadBundleArtifacts(ctx context.Context, ref string, plainHTTP bool, cacheDir string) (*requirement.ArtifactSet, error) {
	credFunc, err := registry.NewCredentialFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to load registry credentials: %w", err)
	}

	repo, err := registry.NewRepository(ref, credFunc, plainHTTP)
	if err != nil {
		return nil, err
	}

	tag := registry.ParseTag(ref)

	// Use on-disk OCI store when cacheDir is set, otherwise fall back to in-memory.
	var store oras.Target
	if cacheDir != "" {
		ociStore, err := getOrCreateOCIStore(cacheDir)
		if err != nil {
			return nil, fmt.Errorf("failed to open cache store: %w", err)
		}
		store = ociStore
	} else {
		store = memory.New()
	}

	_, err = oras.Copy(ctx, repo, tag, store, tag, oras.DefaultCopyOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to pull from registry: %w", err)
	}

	b, err := bundle.Unpack(ctx, store, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack bundle: %w", err)
	}

	result, err := requirement.ClassifyBundle(b)
	if err != nil {
		return nil, fmt.Errorf("failed to classify bundle: %w", err)
	}

	return result, nil
}

// IsOCIReference returns true if the source looks like an OCI reference.
func IsOCIReference(source string) bool {
	// OCI references contain a registry host (domain with optional port)
	// Examples: ghcr.io/org/repo:tag, localhost:5000/repo:tag, http://registry/repo
	return strings.Contains(source, "/") && (strings.Contains(source, ":") || strings.Contains(source, "//"))
}
