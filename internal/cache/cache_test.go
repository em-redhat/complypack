// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveDir(t *testing.T) {
	t.Run("explicit flag overrides all defaults", func(t *testing.T) {
		dir, err := ResolveDir("/tmp/my-cache")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/my-cache", dir)
	})

	t.Run("explicit flag overrides XDG_CACHE_HOME", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "/xdg/cache")
		dir, err := ResolveDir("/explicit/path")
		require.NoError(t, err)
		assert.Equal(t, "/explicit/path", dir)
	})

	t.Run("XDG_CACHE_HOME is respected when no explicit flag", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "/home/user/.cache")
		dir, err := ResolveDir("")
		require.NoError(t, err)
		assert.Equal(t, "/home/user/.cache/complypack", dir)
	})

	t.Run("falls back to HOME/.cache/complypack per XDG spec", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		// HOME is typically always set in test environments
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		dir, err := ResolveDir("")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(homeDir, ".cache", "complypack"), dir)
	})

	t.Run("empty XDG_CACHE_HOME treated as unset", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		dir, err := ResolveDir("")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(homeDir, ".cache", "complypack"), dir)
	})
}

func TestNewOCIStore(t *testing.T) {
	t.Run("creates directory and returns store", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "new-cache")

		store, err := NewOCIStore(dir)
		require.NoError(t, err)
		require.NotNil(t, store)

		// Verify directory was created
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("opens existing directory", func(t *testing.T) {
		dir := t.TempDir()

		store, err := NewOCIStore(dir)
		require.NoError(t, err)
		require.NotNil(t, store)
	})

	t.Run("creates nested parent directories", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "a", "b", "c", "cache")

		store, err := NewOCIStore(dir)
		require.NoError(t, err)
		require.NotNil(t, store)

		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}

func TestClean(t *testing.T) {
	t.Run("removes contents but preserves directory", func(t *testing.T) {
		dir := t.TempDir()

		// Create some files and a subdirectory in the cache
		err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("data"), 0600)
		require.NoError(t, err)
		err = os.MkdirAll(filepath.Join(dir, "subdir"), 0750)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(dir, "subdir", "nested.txt"), []byte("nested"), 0600)
		require.NoError(t, err)

		err = Clean(dir)
		require.NoError(t, err)

		// Verify directory still exists but is empty
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Empty(t, entries, "directory should be empty after clean")
	})

	t.Run("non-existent directory returns nil", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "nonexistent")

		err := Clean(dir)
		require.NoError(t, err)
	})
}
