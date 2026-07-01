// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/complytime/complypack/internal/requirement"
	"github.com/gemaraproj/go-gemara"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testApplicabilityPolicy() *requirement.ResolvedPolicy {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id: "app-catalog",
			ApplicabilityGroups: []gemara.Group{
				{Id: "kubernetes", Title: "Kubernetes Workloads", Description: "Controls for Kubernetes clusters"},
				{Id: "docker", Title: "Docker Containers", Description: "Controls for standalone Docker deployments"},
			},
		},
		Controls: []gemara.Control{
			{
				Id: "CTL-001",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTL-001-AR1", Text: "Both platforms", Applicability: []string{"kubernetes", "docker"}},
					{Id: "CTL-001-AR2", Text: "Kubernetes only", Applicability: []string{"kubernetes"}},
				},
			},
			{
				Id: "CTL-002",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTL-002-AR1", Text: "Docker only", Applicability: []string{"docker"}},
					{Id: "CTL-002-AR2", Text: "No group"},
				},
			},
		},
	}

	policy := &gemara.Policy{
		Metadata: gemara.Metadata{
			Id:                "app-policy",
			MappingReferences: []gemara.MappingReference{{Id: "app-catalog"}},
		},
		Imports: gemara.Imports{
			Catalogs: []gemara.CatalogImport{{ReferenceId: "app-catalog"}},
		},
	}

	set := &requirement.ArtifactSet{
		Catalogs: map[string]*gemara.ControlCatalog{"app-catalog": catalog},
		Policies: map[string]*gemara.Policy{"app-policy": policy},
		Guidance: make(map[string]*gemara.GuidanceCatalog),
	}

	rp, err := requirement.ResolvePolicy(*policy, set)
	if err != nil {
		panic(err)
	}
	return rp
}

func TestHandleGetApplicabilityGroups(t *testing.T) {
	store := &ResourceStore{
		artifacts: map[string]any{},
		resolved: map[string]*requirement.ResolvedPolicy{
			"app-policy": testApplicabilityPolicy(),
		},
		schemas: map[string][]byte{},
	}

	handler := handleGetApplicabilityGroups(store)

	t.Run("returns all groups with requirements", func(t *testing.T) {
		input := map[string]interface{}{"catalogName": "app-policy"}
		inputJSON, err := json.Marshal(input)
		require.NoError(t, err)

		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Arguments: json.RawMessage(inputJSON),
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok)

		var response struct {
			Catalog   string                               `json:"catalog"`
			Groups    []requirement.ApplicabilityGroupInfo `json:"groups"`
			Ungrouped []string                             `json:"ungrouped"`
		}
		err = json.Unmarshal([]byte(textContent.Text), &response)
		require.NoError(t, err)

		assert.Equal(t, "app-policy", response.Catalog)
		assert.Len(t, response.Groups, 2)
		assert.Len(t, response.Ungrouped, 1)
		assert.Contains(t, response.Ungrouped, "CTL-002-AR2")
	})

	t.Run("filter by requirement IDs", func(t *testing.T) {
		input := map[string]interface{}{
			"catalogName":    "app-policy",
			"requirementIds": []string{"CTL-001-AR2"},
		}
		inputJSON, err := json.Marshal(input)
		require.NoError(t, err)

		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Arguments: json.RawMessage(inputJSON),
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)

		textContent := result.Content[0].(*mcp.TextContent)
		var response struct {
			Groups    []requirement.ApplicabilityGroupInfo `json:"groups"`
			Ungrouped []string                             `json:"ungrouped"`
		}
		err = json.Unmarshal([]byte(textContent.Text), &response)
		require.NoError(t, err)

		assert.Len(t, response.Groups, 1)
		assert.Equal(t, "kubernetes", response.Groups[0].ID)
		assert.Empty(t, response.Ungrouped)
	})

	t.Run("catalog name fallback", func(t *testing.T) {
		catalog := &gemara.ControlCatalog{
			Metadata: gemara.Metadata{
				Id: "bare-catalog",
				ApplicabilityGroups: []gemara.Group{
					{Id: "tier-1", Title: "Tier 1", Description: "Essential controls"},
				},
			},
			Controls: []gemara.Control{
				{
					Id: "BC-001",
					AssessmentRequirements: []gemara.AssessmentRequirement{
						{Id: "BC-001-AR1", Text: "Tier 1 req", Applicability: []string{"tier-1"}},
					},
				},
			},
		}
		catalogStore := &ResourceStore{
			artifacts: map[string]any{"bare-catalog": catalog},
			resolved:  map[string]*requirement.ResolvedPolicy{},
			schemas:   map[string][]byte{},
		}
		catalogHandler := handleGetApplicabilityGroups(catalogStore)

		input := map[string]interface{}{"catalogName": "bare-catalog"}
		inputJSON, err := json.Marshal(input)
		require.NoError(t, err)

		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Arguments: json.RawMessage(inputJSON),
			},
		}

		result, err := catalogHandler(context.Background(), req)
		require.NoError(t, err)

		textContent := result.Content[0].(*mcp.TextContent)
		var response struct {
			Groups []requirement.ApplicabilityGroupInfo `json:"groups"`
		}
		err = json.Unmarshal([]byte(textContent.Text), &response)
		require.NoError(t, err)

		assert.Len(t, response.Groups, 1)
		assert.Equal(t, "tier-1", response.Groups[0].ID)
	})

	t.Run("policy not found", func(t *testing.T) {
		input := map[string]interface{}{"catalogName": "nonexistent"}
		inputJSON, err := json.Marshal(input)
		require.NoError(t, err)

		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Arguments: json.RawMessage(inputJSON),
			},
		}

		result, err := handler(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("invalid input", func(t *testing.T) {
		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Arguments: json.RawMessage([]byte(`{invalid`)),
			},
		}

		result, err := handler(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid input")
	})
}

func TestCreateGetApplicabilityGroupsTool(t *testing.T) {
	tool := createGetApplicabilityGroupsTool()

	assert.Equal(t, "get_applicability_groups", tool.Name)
	assert.NotEmpty(t, tool.Description)

	schema, ok := tool.InputSchema.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "object", schema["type"])

	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)

	catalogName, ok := properties["catalogName"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "string", catalogName["type"])

	requirementIds, ok := properties["requirementIds"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "array", requirementIds["type"])

	required, ok := schema["required"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, required, "catalogName")
}
