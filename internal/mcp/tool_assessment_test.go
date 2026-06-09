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

func testEffectivePolicy() *gemara.EffectivePolicy {
	return &gemara.EffectivePolicy{
		Policy: gemara.Policy{
			Metadata: gemara.Metadata{Id: "test-policy"},
			Adherence: gemara.Adherence{
				AssessmentPlans: []gemara.AssessmentPlan{
					{
						RequirementId: "TEST-001-AR1",
						Parameters: []gemara.Parameter{
							{Label: "threshold", AcceptedValues: []string{"90"}},
						},
					},
				},
			},
		},
		ControlCatalogs: []gemara.ControlCatalog{
			{
				Metadata: gemara.Metadata{Id: "test-catalog"},
				Controls: []gemara.Control{
					{
						Id: "TEST-001",
						AssessmentRequirements: []gemara.AssessmentRequirement{
							{
								Id:            "TEST-001-AR1",
								Text:          "Test requirement",
								Applicability: []string{"test"},
							},
							{
								Id:   "TEST-001-AR2",
								Text: "Second requirement",
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
				},
			},
		},
	}
}

func TestHandleGetAssessmentRequirements(t *testing.T) {
	store := &ResourceStore{
		artifacts: map[string]any{},
		effective: map[string]*gemara.EffectivePolicy{
			"test-policy": testEffectivePolicy(),
		},
		schemas: map[string][]byte{},
	}

	handler := handleGetAssessmentRequirements(store)

	t.Run("successful extraction", func(t *testing.T) {
		input := map[string]interface{}{
			"catalogName": "test-policy",
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

		assert.Equal(t, "test-policy", response["catalog"])
		assert.Equal(t, float64(3), response["count"])

		requirements, ok := response["requirements"].([]interface{})
		require.True(t, ok)
		assert.Len(t, requirements, 3)
	})

	t.Run("policy not found", func(t *testing.T) {
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
			"catalogName": "test-policy",
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
		assert.Equal(t, float64(2), response["count"])
	})

	t.Run("parameters from assessment plans", func(t *testing.T) {
		input := map[string]interface{}{
			"catalogName": "test-policy",
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

		requirements := response["requirements"].([]interface{})
		firstReq := requirements[0].(map[string]interface{})
		params := firstReq["parameters"].(map[string]interface{})
		assert.Equal(t, "90", params["threshold"])
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

func TestExtractFromEffectivePolicy(t *testing.T) {
	ep := testEffectivePolicy()

	t.Run("extract all", func(t *testing.T) {
		results := extractFromEffectivePolicy(ep, "")
		assert.Len(t, results, 3)
	})

	t.Run("filter by control", func(t *testing.T) {
		results := extractFromEffectivePolicy(ep, "TEST-001")
		assert.Len(t, results, 2)
		assert.Equal(t, "TEST-001", results[0].ControlID)
		assert.Equal(t, "TEST-001", results[1].ControlID)
	})

	t.Run("parameters populated from assessment plans", func(t *testing.T) {
		results := extractFromEffectivePolicy(ep, "TEST-001")
		assert.Equal(t, "90", results[0].Parameters["threshold"])
		assert.Empty(t, results[1].Parameters)
	})
}
