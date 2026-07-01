// SPDX-License-Identifier: Apache-2.0

package requirement

import "github.com/gemaraproj/go-gemara"

// TriageResult classifies a policy's assessment plans by automation mode.
type TriageResult struct {
	PolicyID                string                 `json:"policy"`
	GlobalEvaluationMethods []EvaluationMethodInfo `json:"global_evaluation_methods"`
	Automated               []TriagedPlan          `json:"automated"`
	Manual                  []TriagedPlan          `json:"manual"`
	Counts                  TriageCounts           `json:"counts"`
}

// EvaluationMethodInfo is a serializable summary of an AcceptedMethod.
type EvaluationMethodInfo struct {
	ID       string       `json:"id"`
	Mode     string       `json:"mode"`
	Executor ExecutorInfo `json:"executor,omitempty"`
}

// ExecutorInfo is a serializable summary of an Actor.
type ExecutorInfo struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// TriagedPlan is an assessment plan classified by automation mode.
type TriagedPlan struct {
	PlanID           string `json:"plan_id"`
	RequirementID    string `json:"requirement_id"`
	EvaluationMethod string `json:"evaluation_method"`
	Executor         string `json:"executor,omitempty"`
}

// TriageCounts summarises the automation split.
type TriageCounts struct {
	Automated int `json:"automated"`
	Manual    int `json:"manual"`
	Total     int `json:"total"`
}

// TriageAssessmentPlans classifies each assessment plan in a resolved
// policy as Automated or Manual based on evaluation methods. Per-plan
// evaluation methods override the global default.
func TriageAssessmentPlans(rp *ResolvedPolicy) *TriageResult {
	policy := rp.Policy
	result := &TriageResult{
		PolicyID:  policy.Metadata.Id,
		Automated: []TriagedPlan{},
		Manual:    []TriagedPlan{},
	}

	for _, m := range policy.Adherence.EvaluationMethods {
		result.GlobalEvaluationMethods = append(result.GlobalEvaluationMethods, toEvaluationMethodInfo(m))
	}

	for _, plan := range policy.Adherence.AssessmentPlans {
		methods := plan.EvaluationMethods
		if len(methods) == 0 {
			methods = policy.Adherence.EvaluationMethods
		}

		automatedMethod, isAutomated := findFirstByMode(methods, gemara.ModeAutomated)

		if isAutomated {
			result.Automated = append(result.Automated, TriagedPlan{
				PlanID:           plan.Id,
				RequirementID:    plan.RequirementId,
				EvaluationMethod: automatedMethod.Id,
				Executor:         automatedMethod.Executor.Id,
			})
		} else {
			tp := TriagedPlan{
				PlanID:        plan.Id,
				RequirementID: plan.RequirementId,
			}
			if manualMethod, ok := findFirstByMode(methods, gemara.ModeManual); ok {
				tp.EvaluationMethod = manualMethod.Id
			}
			result.Manual = append(result.Manual, tp)
		}
	}

	result.Counts = TriageCounts{
		Automated: len(result.Automated),
		Manual:    len(result.Manual),
		Total:     len(result.Automated) + len(result.Manual),
	}

	return result
}

func findFirstByMode(methods []gemara.AcceptedMethod, mode gemara.ModeType) (gemara.AcceptedMethod, bool) {
	for _, m := range methods {
		if m.Mode == mode {
			return m, true
		}
	}
	return gemara.AcceptedMethod{}, false
}

func toEvaluationMethodInfo(m gemara.AcceptedMethod) EvaluationMethodInfo {
	return EvaluationMethodInfo{
		ID:   m.Id,
		Mode: m.Mode.String(),
		Executor: ExecutorInfo{
			ID:   m.Executor.Id,
			Name: m.Executor.Name,
		},
	}
}
