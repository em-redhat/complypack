// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"testing"

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

func TestIsOCISource(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{"oci://ghcr.io/org/catalog:v1", true},
		{"oci+http://localhost:5000/catalog:v1", true},
		{"ghcr.io/org/catalog:v1", true},
		{"localhost:5000/catalog:v1", true},
		{"file:///path/to/catalog.yaml", false},
		{"/local/path/catalog.yaml", false},
		{"catalog.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := isOCISource(tt.source)
			assert.Equal(t, tt.want, got)
		})
	}
}
