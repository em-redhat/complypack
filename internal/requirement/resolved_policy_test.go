// SPDX-License-Identifier: Apache-2.0

package requirement

import (
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvedPolicy_QueryMethods(t *testing.T) {
	set := testArtifactSet()
	policy := set.Policies["test-policy"]

	rp, err := ResolvePolicy(*policy, set)
	require.NoError(t, err)

	t.Run("RequirementsForControl", func(t *testing.T) {
		reqs := rp.RequirementsForControl("CTRL-001")
		assert.Len(t, reqs, 1)
		assert.Equal(t, "REQ-001", reqs[0].Id)
	})

	t.Run("RequirementsForControl unknown", func(t *testing.T) {
		reqs := rp.RequirementsForControl("UNKNOWN")
		assert.Empty(t, reqs)
	})

	t.Run("ControlCatalog", func(t *testing.T) {
		cat := rp.ControlCatalog("test-catalog")
		assert.NotNil(t, cat)
		assert.Equal(t, "test-catalog", cat.Metadata.Id)
	})

	t.Run("ControlCatalog unknown", func(t *testing.T) {
		cat := rp.ControlCatalog("unknown")
		assert.Nil(t, cat)
	})

	t.Run("GuidanceCatalog unknown", func(t *testing.T) {
		gc := rp.GuidanceCatalog("unknown")
		assert.Nil(t, gc)
	})

	t.Run("ControlIDs", func(t *testing.T) {
		ids := rp.ControlIDs()
		assert.Contains(t, ids, "CTRL-001")
	})

	t.Run("ParametersForRequirement", func(t *testing.T) {
		params := rp.ParametersForRequirement("REQ-001")
		assert.Len(t, params, 1)
		assert.Equal(t, "timeout", params[0].Label)
		assert.Equal(t, []string{"30s"}, params[0].AcceptedValues)
	})

	t.Run("ParametersForRequirement unknown", func(t *testing.T) {
		params := rp.ParametersForRequirement("UNKNOWN")
		assert.Empty(t, params)
	})
}

func TestResolvedPolicy_ImportedGuidanceIDs(t *testing.T) {
	guidanceCatalog := &gemara.GuidanceCatalog{
		Metadata: gemara.Metadata{Id: "guidance-1"},
		Guidelines: []gemara.Guideline{
			{Id: "GL-001", Title: "Test guideline"},
		},
	}

	policy := &gemara.Policy{
		Metadata: gemara.Metadata{
			Id: "test-policy",
			MappingReferences: []gemara.MappingReference{
				{Id: "test-catalog"},
				{Id: "guidance-1"},
			},
		},
		Imports: gemara.Imports{
			Catalogs: []gemara.CatalogImport{
				{ReferenceId: "test-catalog"},
			},
			Guidance: []gemara.GuidanceImport{
				{ReferenceId: "guidance-1"},
			},
		},
		Adherence: gemara.Adherence{},
	}

	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "test-catalog"},
		Controls: []gemara.Control{
			{
				Id: "CTRL-001",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "REQ-001", Text: "Verify"},
				},
			},
		},
	}

	set := &ArtifactSet{
		Catalogs: map[string]*gemara.ControlCatalog{"test-catalog": catalog},
		Policies: map[string]*gemara.Policy{"test-policy": policy},
		Guidance: map[string]*gemara.GuidanceCatalog{"guidance-1": guidanceCatalog},
		Mappings: make(map[string]*gemara.MappingDocument),
	}

	rp, err := ResolvePolicy(*policy, set)
	require.NoError(t, err)

	ids := rp.ImportedGuidanceIDs()
	assert.Equal(t, []string{"guidance-1"}, ids)
}
