// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gemaraproj/go-gemara"
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

		var requirements []AssessmentRequirementInfo

		// Check if it's an effective policy (resolved policy graph)
		if ep, found := store.effective[input.CatalogName]; found {
			requirements = extractFromEffectivePolicy(ep, input.ControlID)
		} else if policy, found := store.policies[input.CatalogName]; found {
			// Standalone policy without resolution
			requirements = extractFromPolicy(policy, input.ControlID)
		} else if catalog, found := store.catalogs[input.CatalogName]; found {
			// Standalone control catalog
			requirements = extractFromCatalog(catalog, input.ControlID)
		} else {
			return nil, fmt.Errorf("catalog or policy %q not found", input.CatalogName)
		}

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

// extractFromEffectivePolicy extracts requirements from a resolved policy graph.
func extractFromEffectivePolicy(ep *gemara.EffectivePolicy, filterControlID string) []AssessmentRequirementInfo {
	var results []AssessmentRequirementInfo

	// Build parameter map from assessment plans
	planParams := make(map[string][]gemara.Parameter)
	for _, plan := range ep.Policy.Adherence.AssessmentPlans {
		if len(plan.Parameters) > 0 {
			planParams[plan.RequirementId] = plan.Parameters
		}
	}

	// Iterate resolved catalogs (overlays already applied)
	for _, catalog := range ep.ControlCatalogs {
		for _, control := range catalog.Controls {
			if filterControlID != "" && control.Id != filterControlID {
				continue
			}

			for _, req := range control.AssessmentRequirements {
				info := AssessmentRequirementInfo{
					ID:            req.Id,
					ControlID:     control.Id,
					Text:          req.Text,
					Applicability: req.Applicability,
					Parameters:    make(map[string]string),
				}

				// Add structured parameters from assessment plan
				if params, found := planParams[req.Id]; found {
					for _, param := range params {
						if len(param.AcceptedValues) == 1 {
							info.Parameters[param.Label] = param.AcceptedValues[0]
						} else if len(param.AcceptedValues) > 1 {
							info.Parameters[param.Label] = strings.Join(param.AcceptedValues, ", ")
						}
						if param.Description != "" {
							info.Parameters[param.Label+"_description"] = param.Description
						}
					}
				}

				results = append(results, info)
			}
		}
	}

	return results
}

// extractFromPolicy extracts requirements from a standalone policy (no resolution).
func extractFromPolicy(policy *gemara.Policy, filterControlID string) []AssessmentRequirementInfo {
	var results []AssessmentRequirementInfo

	// Build parameter map from assessment plans
	planParams := make(map[string][]gemara.Parameter)
	for _, plan := range policy.Adherence.AssessmentPlans {
		if len(plan.Parameters) > 0 {
			planParams[plan.RequirementId] = plan.Parameters
		}
	}

	// Iterate policy requirement modifications
	for _, catalogImport := range policy.Imports.Catalogs {
		for _, mod := range catalogImport.AssessmentRequirementModifications {
			if filterControlID != "" && mod.TargetId != filterControlID {
				continue
			}

			info := AssessmentRequirementInfo{
				ID:            mod.Id,
				ControlID:     mod.TargetId,
				Text:          mod.Text,
				Applicability: mod.Applicability,
				Parameters:    make(map[string]string),
			}

			// Add structured parameters from assessment plan
			if params, found := planParams[mod.Id]; found {
				for _, param := range params {
					if len(param.AcceptedValues) == 1 {
						info.Parameters[param.Label] = param.AcceptedValues[0]
					} else if len(param.AcceptedValues) > 1 {
						info.Parameters[param.Label] = strings.Join(param.AcceptedValues, ", ")
					}
					if param.Description != "" {
						info.Parameters[param.Label+"_description"] = param.Description
					}
				}
			}

			results = append(results, info)
		}
	}

	return results
}

// extractFromCatalog extracts requirements from a standalone control catalog.
func extractFromCatalog(catalog *gemara.ControlCatalog, filterControlID string) []AssessmentRequirementInfo {
	var results []AssessmentRequirementInfo

	for _, control := range catalog.Controls {
		if filterControlID != "" && control.Id != filterControlID {
			continue
		}

		for _, req := range control.AssessmentRequirements {
			info := AssessmentRequirementInfo{
				ID:            req.Id,
				ControlID:     control.Id,
				Text:          req.Text,
				Applicability: req.Applicability,
				Parameters:    make(map[string]string),
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
