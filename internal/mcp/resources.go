// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"github.com/complytime/complypack/internal/evaluator"
	"github.com/gemaraproj/go-gemara"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

// ResourceStore manages artifacts and schemas for MCP resource and tool handlers.
type ResourceStore struct {
	artifacts  map[string]any                     // all Gemara artifacts by metadata.id (for ReadResource)
	effective  map[string]*gemara.EffectivePolicy // resolved policy graphs (for assessment tools)
	schemas    map[string][]byte                  // platform schemas (bytes for MCP resources)
	cueSchemas map[string]cue.Value               // compiled CUE schemas (for contract validation)
	evaluators *evaluator.Registry                // available policy evaluators
}

// NewResourceStore creates a ResourceStore.
func NewResourceStore(
	artifacts map[string]any,
	effective map[string]*gemara.EffectivePolicy,
	schemas map[string][]byte,
	cueSchemas map[string]cue.Value,
	evaluators *evaluator.Registry,
) *ResourceStore {
	return &ResourceStore{
		artifacts:  artifacts,
		effective:  effective,
		schemas:    schemas,
		cueSchemas: cueSchemas,
		evaluators: evaluators,
	}
}

// CUESchema returns the compiled CUE schema for a platform.
// Returns an error if the platform has no CUE schema loaded.
func (rs *ResourceStore) CUESchema(platform string) (cue.Value, error) {
	val, ok := rs.cueSchemas[platform]
	if !ok {
		return cue.Value{}, fmt.Errorf("no CUE schema loaded for platform %q", platform)
	}
	return val, nil
}

// ListResources returns all available catalog and schema resources.
func (rs *ResourceStore) ListResources(ctx context.Context) ([]mcp.Resource, error) {
	var resources []mcp.Resource

	for name := range rs.artifacts {
		resources = append(resources, mcp.Resource{
			URI:      fmt.Sprintf("%s://%s/%s", URIScheme, ResourceTypeCatalog, name),
			Name:     fmt.Sprintf("Gemara Artifact: %s", name),
			MIMEType: MIMETypeYAML,
		})
	}

	resources = append(resources, mcp.Resource{
		URI:      fmt.Sprintf("%s://%s", URIScheme, ResourceTypeSchema),
		Name:     "Available Platform Schemas",
		MIMEType: MIMETypeJSON,
	})

	for platform := range rs.schemas {
		mime := MIMETypeCUE
		if isJSONSchema(rs.schemas[platform]) {
			mime = MIMETypeJSONSchema
		}
		resources = append(resources, mcp.Resource{
			URI:      fmt.Sprintf("%s://%s/%s", URIScheme, ResourceTypeSchema, platform),
			Name:     fmt.Sprintf("Platform Schema: %s", platform),
			MIMEType: mime,
		})
	}

	return resources, nil
}

// ReadResource returns the content for a specific resource URI.
func (rs *ResourceStore) ReadResource(ctx context.Context, uri string) ([]*mcp.ResourceContents, error) {
	if !strings.HasPrefix(uri, URIScheme+"://") {
		return nil, fmt.Errorf("invalid URI scheme: expected %s://", URIScheme)
	}

	path := strings.TrimPrefix(uri, URIScheme+"://")
	parts := strings.SplitN(path, "/", 2)

	resourceType := parts[0]

	switch resourceType {
	case ResourceTypeEvaluator:
		return rs.readEvaluatorResource(uri)

	case ResourceTypeCatalog:
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid URI format: %s", uri)
		}
		artifact, ok := rs.artifacts[parts[1]]
		if !ok {
			return nil, fmt.Errorf("artifact %q not found", parts[1])
		}
		data, err := yaml.Marshal(artifact)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal artifact %q: %w", parts[1], err)
		}
		return []*mcp.ResourceContents{{
			URI:      uri,
			MIMEType: MIMETypeYAML,
			Text:     string(data),
		}}, nil

	case ResourceTypeSchema:
		if len(parts) == 1 || parts[1] == "" {
			return rs.readSchemaListResource(uri)
		}
		data, ok := rs.schemas[parts[1]]
		if !ok {
			return nil, fmt.Errorf("schema %q not found", parts[1])
		}
		mime := MIMETypeCUE
		if isJSONSchema(data) {
			mime = MIMETypeJSONSchema
		}
		return []*mcp.ResourceContents{{
			URI:      uri,
			MIMEType: mime,
			Text:     string(data),
		}}, nil

	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

func (rs *ResourceStore) readSchemaListResource(uri string) ([]*mcp.ResourceContents, error) {
	type schemaInfo struct {
		Platform string `json:"platform"`
		Format   string `json:"format"`
	}

	var list []schemaInfo
	for platform := range rs.schemas {
		format := "json-schema"
		if _, hasCUE := rs.cueSchemas[platform]; hasCUE {
			if _, hasJSON := rs.schemas[platform]; hasJSON && isJSONSchema(rs.schemas[platform]) {
				format = "json-schema"
			} else {
				format = "cue"
			}
		}
		list = append(list, schemaInfo{Platform: platform, Format: format})
	}

	data, err := json.Marshal(map[string]interface{}{
		"platforms": list,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema list: %w", err)
	}

	return []*mcp.ResourceContents{{
		URI:      uri,
		MIMEType: MIMETypeJSON,
		Text:     string(data),
	}}, nil
}

func (rs *ResourceStore) readEvaluatorResource(uri string) ([]*mcp.ResourceContents, error) {
	if rs.evaluators == nil {
		return nil, fmt.Errorf("no evaluators available")
	}

	type evalInfo struct {
		ID            string `json:"id"`
		FileExtension string `json:"file_extension"`
	}

	var evals []evalInfo
	for _, id := range rs.evaluators.IDs() {
		e, _ := rs.evaluators.Get(id)
		evals = append(evals, evalInfo{
			ID:            e.ID(),
			FileExtension: e.FileExtension(),
		})
	}

	data, err := json.Marshal(map[string]interface{}{
		"evaluators": evals,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal evaluators: %w", err)
	}

	return []*mcp.ResourceContents{{
		URI:      uri,
		MIMEType: MIMETypeJSON,
		Text:     string(data),
	}}, nil
}

// isJSONSchema returns true if the data looks like JSON (starts with '{').
func isJSONSchema(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}
