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

func testDeltaStore() (*ResourceStore, *requirement.ArtifactSet) {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "container-baseline"},
		Controls: []gemara.Control{
			{
				Id:    "CTL-TLS-001",
				Title: "TLS Configuration",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTL-TLS-001-AR1", Text: "TLS minimum version"},
				},
			},
		},
	}

	policy := &gemara.Policy{
		Metadata: gemara.Metadata{
			Id: "org-policy",
			MappingReferences: []gemara.MappingReference{
				{Id: "container-baseline"},
			},
		},
		Imports: gemara.Imports{
			Catalogs: []gemara.CatalogImport{
				{ReferenceId: "container-baseline"},
			},
		},
		Adherence: gemara.Adherence{
			AssessmentPlans: []gemara.AssessmentPlan{
				{
					RequirementId: "CTL-TLS-001-AR1",
					Parameters: []gemara.Parameter{
						{Label: "tls_minimum_version", AcceptedValues: []string{"1.3"}},
					},
				},
			},
		},
	}

	set := &requirement.ArtifactSet{
		Catalogs: map[string]*gemara.ControlCatalog{"container-baseline": catalog},
		Policies: map[string]*gemara.Policy{"org-policy": policy},
		Guidance: make(map[string]*gemara.GuidanceCatalog),
		Mappings: make(map[string]*gemara.MappingDocument),
	}

	rp, _ := requirement.ResolvePolicy(*policy, set)

	store := &ResourceStore{
		artifacts: map[string]any{"container-baseline": catalog, "org-policy": policy},
		resolved:  map[string]*requirement.ResolvedPolicy{"org-policy": rp},
		schemas:   map[string][]byte{},
	}

	return store, set
}

func TestHandleAnalyzeParameterDelta(t *testing.T) {
	store, set := testDeltaStore()
	handler := handleAnalyzeParameterDelta(store, set)

	t.Run("successful analysis", func(t *testing.T) {
		input := map[string]interface{}{"policyName": "org-policy"}
		inputJSON, err := json.Marshal(input)
		require.NoError(t, err)

		req := &mcp.CallToolRequest{
			Params: &mcp.CallToolParamsRaw{
				Arguments: json.RawMessage(inputJSON),
			},
		}

		result, err := handler(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &response)
		require.NoError(t, err)

		assert.Equal(t, "org-policy", response["policy"])
		comparisons, ok := response["comparisons"].([]interface{})
		require.True(t, ok)
		assert.Len(t, comparisons, 1)

		first := comparisons[0].(map[string]interface{})
		assert.Equal(t, "CTL-TLS-001-AR1", first["requirement_id"])
		assert.Equal(t, "1.3", first["policy_value"])
		assert.Equal(t, "TLS minimum version", first["requirement_text"])
	})

	t.Run("policy not found", func(t *testing.T) {
		input := map[string]interface{}{"policyName": "nonexistent"}
		inputJSON, _ := json.Marshal(input)
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
	})
}

func TestCreateAnalyzeParameterDeltaTool(t *testing.T) {
	tool := createAnalyzeParameterDeltaTool()
	assert.Equal(t, "analyze_parameter_delta", tool.Name)
	assert.NotEmpty(t, tool.Description)

	schema, ok := tool.InputSchema.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "object", schema["type"])

	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)
	_, ok = props["policyName"]
	assert.True(t, ok)
}
