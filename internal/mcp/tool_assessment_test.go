// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleGetAssessmentRequirements(t *testing.T) {
	// Create minimal test catalog
	catalog := &gemara.ControlCatalog{}
	catalog.Metadata.Id = "test-catalog"
	catalog.Controls = []gemara.Control{
		{
			Id: "TEST-001",
			AssessmentRequirements: []gemara.AssessmentRequirement{
				{
					Id:            "TEST-001-AR1",
					Text:          "Test requirement",
					Applicability: []string{"test"},
				},
			},
		},
	}

	store := &ResourceStore{
		rawCatalogs: map[string][]byte{
			"test-catalog": []byte("raw yaml"),
		},
		catalogs: map[string]*gemara.ControlCatalog{
			"test-catalog": catalog,
		},
		policies:  map[string]*gemara.Policy{},
		effective: map[string]*gemara.EffectivePolicy{},
		schemas:   map[string][]byte{},
	}

	handler := handleGetAssessmentRequirements(store)

	t.Run("successful extraction", func(t *testing.T) {
		input := map[string]interface{}{
			"catalogName": "test-catalog",
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
		require.NotNil(t, result)
		require.Len(t, result.Content, 1)

		textContent, ok := result.Content[0].(*mcp.TextContent)
		require.True(t, ok)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &response)
		require.NoError(t, err)

		assert.Equal(t, "test-catalog", response["catalog"])
		assert.Equal(t, float64(1), response["count"])

		requirements, ok := response["requirements"].([]interface{})
		require.True(t, ok)
		assert.Len(t, requirements, 1)
	})

	t.Run("catalog not found", func(t *testing.T) {
		input := map[string]interface{}{
			"catalogName": "nonexistent",
		}
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
				Arguments: json.RawMessage([]byte(`{invalid json`)),
			},
		}

		result, err := handler(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid input")
	})

	t.Run("filter by control ID", func(t *testing.T) {
		input := map[string]interface{}{
			"catalogName": "test-catalog",
			"controlId":   "TEST-001",
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
		var response map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &response)
		require.NoError(t, err)

		assert.Equal(t, "TEST-001", response["control_id"])
	})
}

func TestCreateGetAssessmentRequirementsTool(t *testing.T) {
	tool := createGetAssessmentRequirementsTool()

	assert.Equal(t, "get_assessment_requirements", tool.Name)
	assert.NotEmpty(t, tool.Description)

	schema, ok := tool.InputSchema.(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, "object", schema["type"])

	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)

	catalogName, ok := properties["catalogName"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "string", catalogName["type"])

	controlId, ok := properties["controlId"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "string", controlId["type"])

	required, ok := schema["required"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, required, "catalogName")
}

func TestExtractFromCatalog(t *testing.T) {
	catalog := &gemara.ControlCatalog{}
	catalog.Metadata.Id = "test-catalog"
	catalog.Controls = []gemara.Control{
		{
			Id: "TEST-001",
			AssessmentRequirements: []gemara.AssessmentRequirement{
				{
					Id:            "TEST-001-AR1",
					Text:          "First requirement",
					Applicability: []string{"env-a"},
				},
				{
					Id:            "TEST-001-AR2",
					Text:          "Second requirement",
					Applicability: []string{"env-b"},
				},
			},
		},
		{
			Id: "TEST-002",
			AssessmentRequirements: []gemara.AssessmentRequirement{
				{
					Id:   "TEST-002-AR1",
					Text: "Third requirement",
				},
			},
		},
	}

	t.Run("extract all", func(t *testing.T) {
		results := extractFromCatalog(catalog, "")
		assert.Len(t, results, 3)
	})

	t.Run("filter by control", func(t *testing.T) {
		results := extractFromCatalog(catalog, "TEST-001")
		assert.Len(t, results, 2)
		assert.Equal(t, "TEST-001", results[0].ControlID)
		assert.Equal(t, "TEST-001", results[1].ControlID)
	})

	t.Run("no parameters for standalone catalog", func(t *testing.T) {
		results := extractFromCatalog(catalog, "")
		for _, r := range results {
			assert.Empty(t, r.Parameters)
		}
	})
}

func TestExtractFromEffectivePolicy(t *testing.T) {
	ep := &gemara.EffectivePolicy{}
	ep.Policy.Adherence.AssessmentPlans = []gemara.AssessmentPlan{
		{
			RequirementId: "REQ-001",
			Parameters: []gemara.Parameter{
				{
					Label:          "timeout",
					Description:    "Timeout value",
					AcceptedValues: []string{"60"},
				},
			},
		},
	}
	ep.ControlCatalogs = []gemara.ControlCatalog{
		{
			Controls: []gemara.Control{
				{
					Id: "CTRL-001",
					AssessmentRequirements: []gemara.AssessmentRequirement{
						{
							Id:   "REQ-001",
							Text: "Test requirement",
						},
					},
				},
			},
		},
	}

	t.Run("extracts requirements with parameters", func(t *testing.T) {
		results := extractFromEffectivePolicy(ep, "")
		assert.Len(t, results, 1)
		assert.Equal(t, "REQ-001", results[0].ID)
		assert.Equal(t, "60", results[0].Parameters["timeout"])
		assert.Equal(t, "Timeout value", results[0].Parameters["timeout_description"])
	})

	t.Run("filters by control ID", func(t *testing.T) {
		results := extractFromEffectivePolicy(ep, "CTRL-001")
		assert.Len(t, results, 1)

		results = extractFromEffectivePolicy(ep, "NONEXISTENT")
		assert.Empty(t, results)
	})
}
