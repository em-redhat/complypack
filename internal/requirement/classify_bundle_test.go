// SPDX-License-Identifier: Apache-2.0

package requirement

import (
	"testing"

	"github.com/gemaraproj/go-gemara/bundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyBundle(t *testing.T) {
	t.Run("policy with catalog import", func(t *testing.T) {
		b := &bundle.Bundle{
			Source: bundle.File{
				Name: "policy.yaml", Type: "Policy", Data: []byte(testPolicyYAML),
			},
			Imports: []bundle.File{
				{Name: "catalog.yaml", Type: "ControlCatalog", Data: []byte(testControlCatalogYAML)},
			},
		}

		result, err := ClassifyBundle(b)
		require.NoError(t, err)
		assert.Len(t, result.Policies, 1)
		assert.Len(t, result.Catalogs, 1)
		assert.Contains(t, result.Policies, "test-policy")
		assert.Contains(t, result.Catalogs, "test-catalog")
	})

	t.Run("empty bundle", func(t *testing.T) {
		b := &bundle.Bundle{}
		_, err := ClassifyBundle(b)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no primary files")
	})

	t.Run("catalog only", func(t *testing.T) {
		b := &bundle.Bundle{
			Source: bundle.File{
				Name: "catalog.yaml", Type: "ControlCatalog", Data: []byte(testControlCatalogYAML),
			},
		}

		result, err := ClassifyBundle(b)
		require.NoError(t, err)
		assert.Len(t, result.Catalogs, 1)
		assert.Empty(t, result.Policies)
	})
}
