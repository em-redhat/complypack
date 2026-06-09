// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testArtifacts() map[string]any {
	return map[string]any{
		"controls-v1": &gemara.ControlCatalog{Metadata: gemara.Metadata{Id: "controls-v1"}},
		"security-v2": &gemara.ControlCatalog{Metadata: gemara.Metadata{Id: "security-v2"}},
	}
}

func TestResourceStore_ListResources(t *testing.T) {
	schemas := map[string][]byte{
		"kubernetes": []byte(`{"components": {}}`),
		"terraform":  []byte(`{"components": {}}`),
	}

	store := NewResourceStore(testArtifacts(), nil, schemas, nil, nil)
	resources, err := store.ListResources(context.Background())
	require.NoError(t, err)

	assert.Len(t, resources, 5, "should have 2 artifacts + 1 schema list + 2 schemas")

	catalogURIs := []string{}
	for _, r := range resources {
		if r.MIMEType == MIMETypeYAML {
			catalogURIs = append(catalogURIs, r.URI)
		}
	}
	assert.Contains(t, catalogURIs, "complypack://catalog/controls-v1")
	assert.Contains(t, catalogURIs, "complypack://catalog/security-v2")

	var hasSchemaList bool
	for _, r := range resources {
		if r.URI == "complypack://schema" {
			hasSchemaList = true
			assert.Equal(t, MIMETypeJSON, r.MIMEType)
		}
	}
	assert.True(t, hasSchemaList, "should have schema list resource")

	schemaURIs := []string{}
	for _, r := range resources {
		if r.MIMEType == MIMETypeJSONSchema {
			schemaURIs = append(schemaURIs, r.URI)
		}
	}
	assert.Contains(t, schemaURIs, "complypack://schema/kubernetes")
	assert.Contains(t, schemaURIs, "complypack://schema/terraform")
}

func TestResourceStore_ReadResource(t *testing.T) {
	artifacts := map[string]any{
		"controls-v1": &gemara.ControlCatalog{Metadata: gemara.Metadata{Id: "controls-v1"}},
	}

	schemas := map[string][]byte{
		"kubernetes": []byte(`{"components": {}}`),
	}

	store := NewResourceStore(artifacts, nil, schemas, nil, nil)

	t.Run("read artifact", func(t *testing.T) {
		contents, err := store.ReadResource(context.Background(), "complypack://catalog/controls-v1")
		require.NoError(t, err)
		require.Len(t, contents, 1)
		assert.Equal(t, MIMETypeYAML, contents[0].MIMEType)
		assert.Contains(t, contents[0].Text, "controls-v1")
	})

	t.Run("read schema", func(t *testing.T) {
		contents, err := store.ReadResource(context.Background(), "complypack://schema/kubernetes")
		require.NoError(t, err)
		require.Len(t, contents, 1)
		assert.Equal(t, MIMETypeJSONSchema, contents[0].MIMEType)
		assert.Equal(t, `{"components": {}}`, contents[0].Text)
	})

	t.Run("schema list resource", func(t *testing.T) {
		contents, err := store.ReadResource(context.Background(), "complypack://schema")
		require.NoError(t, err)
		require.Len(t, contents, 1)
		assert.Equal(t, MIMETypeJSON, contents[0].MIMEType)
		assert.Contains(t, contents[0].Text, "kubernetes")
	})

	t.Run("unknown catalog returns error", func(t *testing.T) {
		_, err := store.ReadResource(context.Background(), "complypack://catalog/unknown")
		assert.Error(t, err)
	})

	t.Run("unknown schema returns error", func(t *testing.T) {
		_, err := store.ReadResource(context.Background(), "complypack://schema/unknown")
		assert.Error(t, err)
	})

	t.Run("invalid URI returns error", func(t *testing.T) {
		_, err := store.ReadResource(context.Background(), "invalid://foo/bar")
		assert.Error(t, err)
	})
}
