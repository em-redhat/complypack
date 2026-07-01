// SPDX-License-Identifier: Apache-2.0

package requirement

import (
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
)

func TestTriageAssessmentPlans(t *testing.T) {
	t.Run("all automated via global default", func(t *testing.T) {
		rp := &ResolvedPolicy{
			Policy: gemara.Policy{
				Metadata: gemara.Metadata{Id: "test"},
				Adherence: gemara.Adherence{
					EvaluationMethods: []gemara.AcceptedMethod{
						{Id: "auto", Mode: gemara.ModeAutomated, Executor: gemara.Actor{Id: "opa"}},
					},
					AssessmentPlans: []gemara.AssessmentPlan{
						{Id: "p1", RequirementId: "R1"},
						{Id: "p2", RequirementId: "R2"},
					},
				},
			},
		}

		result := TriageAssessmentPlans(rp)
		assert.Equal(t, 2, result.Counts.Automated)
		assert.Equal(t, 0, result.Counts.Manual)
		assert.Len(t, result.Automated, 2)
		assert.Empty(t, result.Manual)
	})

	t.Run("per-plan override to manual", func(t *testing.T) {
		rp := &ResolvedPolicy{
			Policy: gemara.Policy{
				Metadata: gemara.Metadata{Id: "test"},
				Adherence: gemara.Adherence{
					EvaluationMethods: []gemara.AcceptedMethod{
						{Id: "auto", Mode: gemara.ModeAutomated, Executor: gemara.Actor{Id: "opa"}},
					},
					AssessmentPlans: []gemara.AssessmentPlan{
						{Id: "p1", RequirementId: "R1"},
						{
							Id:            "p2",
							RequirementId: "R2",
							EvaluationMethods: []gemara.AcceptedMethod{
								{Id: "manual-check", Mode: gemara.ModeManual},
							},
						},
					},
				},
			},
		}

		result := TriageAssessmentPlans(rp)
		assert.Equal(t, 1, result.Counts.Automated)
		assert.Equal(t, 1, result.Counts.Manual)
		assert.Equal(t, "R1", result.Automated[0].RequirementID)
		assert.Equal(t, "R2", result.Manual[0].RequirementID)
		assert.Equal(t, "manual-check", result.Manual[0].EvaluationMethod)
	})

	t.Run("no evaluation methods anywhere", func(t *testing.T) {
		rp := &ResolvedPolicy{
			Policy: gemara.Policy{
				Metadata: gemara.Metadata{Id: "test"},
				Adherence: gemara.Adherence{
					AssessmentPlans: []gemara.AssessmentPlan{
						{Id: "p1", RequirementId: "R1"},
					},
				},
			},
		}

		result := TriageAssessmentPlans(rp)
		assert.Equal(t, 0, result.Counts.Automated)
		assert.Equal(t, 1, result.Counts.Manual)
	})

	t.Run("no assessment plans", func(t *testing.T) {
		rp := &ResolvedPolicy{
			Policy: gemara.Policy{
				Metadata: gemara.Metadata{Id: "test"},
				Adherence: gemara.Adherence{
					EvaluationMethods: []gemara.AcceptedMethod{
						{Id: "auto", Mode: gemara.ModeAutomated},
					},
				},
			},
		}

		result := TriageAssessmentPlans(rp)
		assert.Equal(t, 0, result.Counts.Total)
		assert.Empty(t, result.Automated)
		assert.Empty(t, result.Manual)
	})

	t.Run("per-plan override to automated", func(t *testing.T) {
		rp := &ResolvedPolicy{
			Policy: gemara.Policy{
				Metadata: gemara.Metadata{Id: "test"},
				Adherence: gemara.Adherence{
					EvaluationMethods: []gemara.AcceptedMethod{
						{Id: "global-manual", Mode: gemara.ModeManual},
					},
					AssessmentPlans: []gemara.AssessmentPlan{
						{
							Id:            "p1",
							RequirementId: "R1",
							EvaluationMethods: []gemara.AcceptedMethod{
								{Id: "special-auto", Mode: gemara.ModeAutomated, Executor: gemara.Actor{Id: "custom-tool"}},
							},
						},
						{Id: "p2", RequirementId: "R2"},
					},
				},
			},
		}

		result := TriageAssessmentPlans(rp)
		assert.Equal(t, 1, result.Counts.Automated)
		assert.Equal(t, 1, result.Counts.Manual)
		assert.Equal(t, "special-auto", result.Automated[0].EvaluationMethod)
		assert.Equal(t, "custom-tool", result.Automated[0].Executor)
		assert.Equal(t, "global-manual", result.Manual[0].EvaluationMethod)
	})

	t.Run("global evaluation methods populated", func(t *testing.T) {
		rp := &ResolvedPolicy{
			Policy: gemara.Policy{
				Metadata: gemara.Metadata{Id: "test"},
				Adherence: gemara.Adherence{
					EvaluationMethods: []gemara.AcceptedMethod{
						{Id: "eval-1", Mode: gemara.ModeAutomated, Executor: gemara.Actor{Id: "opa", Name: "OPA/Conftest"}},
					},
				},
			},
		}

		result := TriageAssessmentPlans(rp)
		assert.Len(t, result.GlobalEvaluationMethods, 1)
		assert.Equal(t, "eval-1", result.GlobalEvaluationMethods[0].ID)
		assert.Equal(t, "Automated", result.GlobalEvaluationMethods[0].Mode)
		assert.Equal(t, "opa", result.GlobalEvaluationMethods[0].Executor.ID)
		assert.Equal(t, "OPA/Conftest", result.GlobalEvaluationMethods[0].Executor.Name)
	})
}
