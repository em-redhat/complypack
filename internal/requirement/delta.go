// SPDX-License-Identifier: Apache-2.0

package requirement

import "fmt"

// ParameterComparison pairs a structured L3 parameter with the
// L1/L2 requirement text it maps to. The caller interprets the
// relationship — the engine does not judge.
type ParameterComparison struct {
	// RequirementID is the assessment requirement this comparison targets (e.g. "CTL-TLS-001-AR1").
	RequirementID string `json:"requirement_id"`
	// Label is the parameter name from the policy's assessment plan (e.g. "tls_minimum_version").
	Label string `json:"label"`
	// PolicyValue is the concrete value the L3 policy sets for this parameter.
	PolicyValue string `json:"policy_value"`
	// PolicySource is the ID of the policy that provides the parameter value.
	PolicySource string `json:"policy_source"`
	// RequirementText is the L1/L2 assessment requirement text from the catalog.
	RequirementText string `json:"requirement_text"`
	// CatalogSource is the ID of the catalog that defines the requirement.
	CatalogSource string `json:"catalog_source"`
}

// DeltaReport is the result of gathering parameter comparisons
// across a resolved policy.
type DeltaReport struct {
	PolicyID         string                `json:"policy"`
	CatalogsCompared []string              `json:"catalogs_compared"`
	Comparisons      []ParameterComparison `json:"comparisons"`
}

// AnalyzeDelta gathers L3 parameter values alongside the L1/L2
// requirement text they map to. Returns structured pairs for the
// caller to interpret — no verdicts or heuristics.
func AnalyzeDelta(rp *ResolvedPolicy, set *ArtifactSet) (*DeltaReport, error) {
	if rp == nil {
		return nil, fmt.Errorf("resolved policy is nil")
	}

	var catalogIDs []string
	for _, cat := range rp.ControlCatalogs {
		catalogIDs = append(catalogIDs, cat.Metadata.Id)
	}

	reqTextIndex := buildRequirementTextIndex(rp)

	var comparisons []ParameterComparison
	for _, plan := range rp.Policy.Adherence.AssessmentPlans {
		for _, param := range plan.Parameters {
			policyValue := ""
			if len(param.AcceptedValues) > 0 {
				policyValue = param.AcceptedValues[0]
			}

			reqText, catalogSource := reqTextIndex.lookup(plan.RequirementId)

			comparisons = append(comparisons, ParameterComparison{
				RequirementID:   plan.RequirementId,
				Label:           param.Label,
				PolicyValue:     policyValue,
				PolicySource:    rp.Policy.Metadata.Id,
				RequirementText: reqText,
				CatalogSource:   catalogSource,
			})
		}
	}

	return &DeltaReport{
		PolicyID:         rp.Policy.Metadata.Id,
		CatalogsCompared: catalogIDs,
		Comparisons:      comparisons,
	}, nil
}

type requirementTextIndex struct {
	texts   map[string]string
	sources map[string]string
}

func buildRequirementTextIndex(rp *ResolvedPolicy) requirementTextIndex {
	idx := requirementTextIndex{
		texts:   make(map[string]string),
		sources: make(map[string]string),
	}
	for _, cat := range rp.ControlCatalogs {
		for _, ctrl := range cat.Controls {
			for _, ar := range ctrl.AssessmentRequirements {
				idx.texts[ar.Id] = ar.Text
				idx.sources[ar.Id] = cat.Metadata.Id
			}
		}
	}
	for _, gc := range rp.GuidanceCatalogs {
		for _, gl := range gc.Guidelines {
			idx.texts[gl.Id] = gl.Objective
			idx.sources[gl.Id] = gc.Metadata.Id
		}
	}
	return idx
}

func (idx requirementTextIndex) lookup(reqID string) (text, source string) {
	return idx.texts[reqID], idx.sources[reqID]
}
