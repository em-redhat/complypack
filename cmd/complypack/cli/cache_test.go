// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheCommand(t *testing.T) {
	root := New()

	// Find the cache command
	cacheCmd, _, err := root.Find([]string{"cache"})
	require.NoError(t, err, "cache command should exist")
	assert.Equal(t, "cache", cacheCmd.Name())
	assert.NotEmpty(t, cacheCmd.Short, "cache command should have a short description")

	// Find the clean subcommand
	cleanCmd, _, err := cacheCmd.Find([]string{"clean"})
	require.NoError(t, err, "cache clean command should exist")
	assert.Equal(t, "clean", cleanCmd.Name())
	assert.NotEmpty(t, cleanCmd.Short, "clean command should have a short description")

	// Check flags exist
	flags := cleanCmd.Flags()
	assert.NotNil(t, flags.Lookup("cache-dir"), "should have --cache-dir flag")
}

func TestCacheCleanNonExistentDir(t *testing.T) {
	root := New()
	root.SetArgs([]string{"cache", "clean", "--cache-dir", t.TempDir() + "/nonexistent"})
	err := root.Execute()
	require.NoError(t, err, "cleaning non-existent cache should not error")
}

func TestCacheCleanExistingDir(t *testing.T) {
	dir := t.TempDir()
	root := New()
	root.SetArgs([]string{"cache", "clean", "--cache-dir", dir})
	err := root.Execute()
	require.NoError(t, err, "cleaning existing cache should not error")
}
