// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/complytime/complypack/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullCommand(t *testing.T) {
	root := New()

	// Find the pull command
	pullCmd, _, err := root.Find([]string{"pull"})
	require.NoError(t, err, "pull command should exist")
	assert.Equal(t, "pull", pullCmd.Name())
	assert.NotEmpty(t, pullCmd.Short, "pull command should have a short description")

	// Check flags exist
	flags := pullCmd.Flags()
	assert.NotNil(t, flags.Lookup("config"), "should have --config flag")
	assert.NotNil(t, flags.Lookup("cache-dir"), "should have --cache-dir flag")
}

func TestPullSources_SkipsFileSource(t *testing.T) {
	ctx := context.Background()
	cacheDir := t.TempDir()

	// Create a local catalog file
	catalogPath := filepath.Join(t.TempDir(), "catalog.yaml")
	err := os.WriteFile(catalogPath, []byte(`metadata:
  id: test-catalog
  type: ControlCatalog
  gemara-version: "1.0.0"
controls:
  - id: AC-1
    title: Test
    description: Test control
`), 0600)
	require.NoError(t, err)

	cfg := &config.ComplyPackConfig{
		Gemara: config.GemaraConfig{
			Sources: []config.GemaraSourceEntry{
				{Source: catalogPath},
			},
		},
	}

	result, err := pullSources(ctx, cfg, cacheDir)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Pulled)
	assert.Equal(t, 1, result.Skipped)
	assert.Empty(t, result.Errors)
}

func TestPullSources_NoSources(t *testing.T) {
	ctx := context.Background()
	cfg := &config.ComplyPackConfig{}

	_, err := pullSources(ctx, cfg, t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no gemara sources configured")
}
