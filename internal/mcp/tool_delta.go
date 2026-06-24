// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/complytime/complypack/internal/requirement"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func createAnalyzeParameterDeltaTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "analyze_parameter_delta",
		Description: "Gather parameter comparisons across a resolved policy. Returns structured L3 parameter values alongside the L1/L2 requirement text they map to. The caller interprets the relationship — the tool does not judge.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"policyName": map[string]interface{}{
					"type":        "string",
					"description": "Name of the resolved policy to analyze",
				},
			},
			"required": []interface{}{"policyName"},
		},
	}
}

func handleAnalyzeParameterDelta(store *ResourceStore, artifactSet *requirement.ArtifactSet) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input struct {
			PolicyName string `json:"policyName"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}

		rp, found := store.resolved[input.PolicyName]
		if !found {
			return nil, fmt.Errorf("policy %q not found", input.PolicyName)
		}

		report, err := requirement.AnalyzeDelta(rp, artifactSet)
		if err != nil {
			return nil, fmt.Errorf("delta analysis failed: %w", err)
		}

		responseData, err := json.Marshal(report)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: string(responseData),
				},
			},
		}, nil
	}
}
