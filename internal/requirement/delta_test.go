// SPDX-License-Identifier: Apache-2.0

package requirement

import (
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDeltaArtifactSet() *ArtifactSet {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "container-baseline"},
		Controls: []gemara.Control{
			{
				Id:    "CTL-TLS-001",
				Title: "TLS Configuration",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTL-TLS-001-AR1", Text: "TLS minimum version must be enforced"},
				},
			},
			{
				Id:    "CTL-CERT-001",
				Title: "Certificate Management",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTL-CERT-001-AR1", Text: "Certificate validity must not exceed maximum"},
				},
			},
		},
	}

	policy := &gemara.Policy{
		Metadata: gemara.Metadata{
			Id: "org-parent-policy",
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
				{
					RequirementId: "CTL-CERT-001-AR1",
					Parameters: []gemara.Parameter{
						{Label: "max_validity_days", AcceptedValues: []string{"90"}},
					},
				},
			},
		},
	}

	return &ArtifactSet{
		Catalogs: map[string]*gemara.ControlCatalog{"container-baseline": catalog},
		Policies: map[string]*gemara.Policy{"org-parent-policy": policy},
		Guidance: make(map[string]*gemara.GuidanceCatalog),
		Mappings: make(map[string]*gemara.MappingDocument),
	}
}

func TestAnalyzeDelta(t *testing.T) {
	set := testDeltaArtifactSet()
	policy := set.Policies["org-parent-policy"]

	rp, err := ResolvePolicy(*policy, set)
	require.NoError(t, err)

	report, err := AnalyzeDelta(rp, set)
	require.NoError(t, err)

	assert.Equal(t, "org-parent-policy", report.PolicyID)
	assert.Contains(t, report.CatalogsCompared, "container-baseline")
	assert.Len(t, report.Comparisons, 2)

	tls := report.Comparisons[0]
	assert.Equal(t, "CTL-TLS-001-AR1", tls.RequirementID)
	assert.Equal(t, "tls_minimum_version", tls.Label)
	assert.Equal(t, "1.3", tls.PolicyValue)
	assert.Equal(t, "org-parent-policy", tls.PolicySource)
	assert.Equal(t, "TLS minimum version must be enforced", tls.RequirementText)
	assert.Equal(t, "container-baseline", tls.CatalogSource)

	cert := report.Comparisons[1]
	assert.Equal(t, "CTL-CERT-001-AR1", cert.RequirementID)
	assert.Equal(t, "90", cert.PolicyValue)
	assert.Equal(t, "Certificate validity must not exceed maximum", cert.RequirementText)
}

func TestAnalyzeDelta_NilPolicy(t *testing.T) {
	set := testDeltaArtifactSet()
	_, err := AnalyzeDelta(nil, set)
	assert.Error(t, err)
}
