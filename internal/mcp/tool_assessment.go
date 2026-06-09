// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/complytime/complypack/internal/requirement"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// createGetAssessmentRequirementsTool creates the MCP tool definition.
func createGetAssessmentRequirementsTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_assessment_requirements",
		Description: "Extract assessment requirements from a policy or catalog with structured parameters from assessment plans",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"catalogName": map[string]interface{}{
					"type":        "string",
					"description": "Name of the catalog or policy to extract from (e.g., 'my-policy')",
				},
				"controlId": map[string]interface{}{
					"type":        "string",
					"description": "Optional: Specific control ID to filter requirements (e.g., 'CTRL-001')",
				},
			},
			"required": []interface{}{"catalogName"},
		},
	}
}

// AssessmentRequirementInfo contains assessment requirement data with parameters.
type AssessmentRequirementInfo struct {
	ID            string            `json:"id"`
	ControlID     string            `json:"control_id"`
	Text          string            `json:"text"`
	Applicability []string          `json:"applicability,omitempty"`
	Parameters    map[string]string `json:"parameters,omitempty"`
}

// handleGetAssessmentRequirements extracts assessment requirements from a policy or catalog.
func handleGetAssessmentRequirements(store *ResourceStore) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse input
		var input struct {
			CatalogName string `json:"catalogName"`
			ControlID   string `json:"controlId"`
		}

		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return nil, fmt.Errorf("invalid input: %w", err)
		}

		rp, found := store.resolved[input.CatalogName]
		if !found {
			return nil, fmt.Errorf("policy %q not found", input.CatalogName)
		}
		requirements := extractFromResolvedPolicy(rp, input.ControlID)

		// Build response
		responseData, err := json.Marshal(map[string]interface{}{
			"catalog":      input.CatalogName,
			"control_id":   input.ControlID,
			"count":        len(requirements),
			"requirements": requirements,
		})
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

// extractFromResolvedPolicy extracts requirements from a resolved policy graph.
func extractFromResolvedPolicy(rp *requirement.ResolvedPolicy, filterControlID string) []AssessmentRequirementInfo {
	var results []AssessmentRequirementInfo

	controlIDs := rp.ControlIDs()
	if filterControlID != "" {
		controlIDs = []string{filterControlID}
	}

	for _, controlID := range controlIDs {
		for _, req := range rp.RequirementsForControl(controlID) {
			info := AssessmentRequirementInfo{
				ID:            req.Id,
				ControlID:     controlID,
				Text:          req.Text,
				Applicability: req.Applicability,
				Parameters:    make(map[string]string),
			}

			for _, param := range rp.ParametersForRequirement(req.Id) {
				if len(param.AcceptedValues) == 1 {
					info.Parameters[param.Label] = param.AcceptedValues[0]
				} else if len(param.AcceptedValues) > 1 {
					info.Parameters[param.Label] = strings.Join(param.AcceptedValues, ", ")
				}
				if param.Description != "" {
					info.Parameters[param.Label+"_description"] = param.Description
				}
			}

			results = append(results, info)
		}
	}

	return results
}

// GetAssessmentRequirementsHandler returns the handler (for testing).
func GetAssessmentRequirementsHandler(store *ResourceStore) mcp.ToolHandler {
	return handleGetAssessmentRequirements(store)
}
