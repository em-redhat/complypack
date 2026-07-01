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

func testTriagePolicy() *requirement.ResolvedPolicy {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "triage-catalog"},
		Controls: []gemara.Control{
			{
				Id: "CTL-001",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTL-001-AR1", Text: "Automated requirement"},
				},
			},
		},
	}

	policy := &gemara.Policy{
		Metadata: gemara.Metadata{
			Id:                "triage-policy",
			MappingReferences: []gemara.MappingReference{{Id: "triage-catalog"}},
		},
		Imports: gemara.Imports{
			Catalogs: []gemara.CatalogImport{{ReferenceId: "triage-catalog"}},
		},
		Adherence: gemara.Adherence{
			EvaluationMethods: []gemara.AcceptedMethod{
				{Id: "default-eval", Mode: gemara.ModeAutomated, Executor: gemara.Actor{Id: "opa"}},
			},
			AssessmentPlans: []gemara.AssessmentPlan{
				{Id: "ap-1", RequirementId: "CTL-001-AR1"},
			},
		},
	}

	set := &requirement.ArtifactSet{
		Catalogs: map[string]*gemara.ControlCatalog{"triage-catalog": catalog},
		Policies: map[string]*gemara.Policy{"triage-policy": policy},
		Guidance: make(map[string]*gemara.GuidanceCatalog),
	}

	rp, err := requirement.ResolvePolicy(*policy, set)
	if err != nil {
		panic(err)
	}
	return rp
}

func TestHandleGetAutomationTriage(t *testing.T) {
	store := &ResourceStore{
		artifacts: map[string]any{},
		resolved: map[string]*requirement.ResolvedPolicy{
			"triage-policy": testTriagePolicy(),
		},
		schemas: map[string][]byte{},
	}

	handler := handleGetAutomationTriage(store)

	t.Run("returns triage result", func(t *testing.T) {
		input := map[string]interface{}{"policyName": "triage-policy"}
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

		var response requirement.TriageResult
		err = json.Unmarshal([]byte(textContent.Text), &response)
		require.NoError(t, err)

		assert.Equal(t, "triage-policy", response.PolicyID)
		assert.Equal(t, 1, response.Counts.Automated)
		assert.Equal(t, 0, response.Counts.Manual)
	})

	t.Run("policy not found", func(t *testing.T) {
		input := map[string]interface{}{"policyName": "nonexistent"}
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
	})
}

func TestCreateGetAutomationTriageTool(t *testing.T) {
	tool := createGetAutomationTriageTool()

	assert.Equal(t, "get_automation_triage", tool.Name)
	assert.NotEmpty(t, tool.Description)

	schema, ok := tool.InputSchema.(map[string]interface{})
	require.True(t, ok)

	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)

	_, ok = properties["policyName"].(map[string]interface{})
	require.True(t, ok)

	required, ok := schema["required"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, required, "policyName")
}
