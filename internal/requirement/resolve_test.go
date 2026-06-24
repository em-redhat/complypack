// SPDX-License-Identifier: Apache-2.0

package requirement

import (
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testArtifactSet() *ArtifactSet {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "test-catalog"},
		Controls: []gemara.Control{
			{
				Id:    "CTRL-001",
				Title: "Test Control",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{
						Id:            "REQ-001",
						Text:          "Verify the thing",
						Applicability: []string{"kubernetes"},
					},
				},
			},
		},
	}

	policy := &gemara.Policy{
		Metadata: gemara.Metadata{
			Id: "test-policy",
			MappingReferences: []gemara.MappingReference{
				{Id: "test-catalog"},
			},
		},
		Imports: gemara.Imports{
			Catalogs: []gemara.CatalogImport{
				{ReferenceId: "test-catalog"},
			},
		},
		Adherence: gemara.Adherence{
			AssessmentPlans: []gemara.AssessmentPlan{
				{
					RequirementId: "REQ-001",
					Parameters: []gemara.Parameter{
						{
							Label:          "timeout",
							AcceptedValues: []string{"30s"},
						},
					},
				},
			},
		},
	}

	return &ArtifactSet{
		Catalogs: map[string]*gemara.ControlCatalog{"test-catalog": catalog},
		Policies: map[string]*gemara.Policy{"test-policy": policy},
		Guidance: make(map[string]*gemara.GuidanceCatalog),
		Mappings: make(map[string]*gemara.MappingDocument),
	}
}

func TestResolvePolicy(t *testing.T) {
	t.Run("resolves catalog import", func(t *testing.T) {
		set := testArtifactSet()
		policy := set.Policies["test-policy"]

		rp, err := ResolvePolicy(*policy, set)
		require.NoError(t, err)
		assert.Len(t, rp.ControlCatalogs, 1)
		assert.Empty(t, rp.Unresolved)
	})

	t.Run("tracks unresolved imports", func(t *testing.T) {
		set := testArtifactSet()
		policy := set.Policies["test-policy"]
		policy.Metadata.MappingReferences = append(policy.Metadata.MappingReferences,
			gemara.MappingReference{Id: "missing-catalog"})
		policy.Imports.Catalogs = append(policy.Imports.Catalogs,
			gemara.CatalogImport{ReferenceId: "missing-catalog"})

		rp, err := ResolvePolicy(*policy, set)
		require.NoError(t, err)
		assert.Contains(t, rp.Unresolved, "missing-catalog")
		assert.Len(t, rp.ControlCatalogs, 1)
	})

	t.Run("errors on all imports unresolved", func(t *testing.T) {
		set := testArtifactSet()
		delete(set.Catalogs, "test-catalog")
		policy := set.Policies["test-policy"]

		_, err := ResolvePolicy(*policy, set)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no imports could be resolved")
	})

	t.Run("errors on duplicate import refs", func(t *testing.T) {
		set := testArtifactSet()
		policy := set.Policies["test-policy"]
		policy.Imports.Catalogs = append(policy.Imports.Catalogs,
			gemara.CatalogImport{ReferenceId: "test-catalog"})

		_, err := ResolvePolicy(*policy, set)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate catalog import")
	})

	t.Run("errors on duplicate mapping-reference", func(t *testing.T) {
		set := testArtifactSet()
		policy := set.Policies["test-policy"]
		policy.Metadata.MappingReferences = append(policy.Metadata.MappingReferences,
			gemara.MappingReference{Id: "test-catalog"})

		_, err := ResolvePolicy(*policy, set)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate mapping-reference")
	})
}

func TestResolvePolicy_WithExclusions(t *testing.T) {
	set := testArtifactSet()
	set.Catalogs["test-catalog"].Controls = append(set.Catalogs["test-catalog"].Controls,
		gemara.Control{Id: "CTRL-002", Title: "Excluded"})

	policy := set.Policies["test-policy"]
	policy.Imports.Catalogs[0].Exclusions = []string{"CTRL-002"}

	rp, err := ResolvePolicy(*policy, set)
	require.NoError(t, err)
	assert.Len(t, rp.ControlCatalogs[0].Controls, 1)
	assert.Equal(t, "CTRL-001", rp.ControlCatalogs[0].Controls[0].Id)
}

func TestResolvePolicy_WithExtends(t *testing.T) {
	base := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "base-catalog"},
		Controls: []gemara.Control{
			{Id: "BASE-001", Title: "Base Control"},
		},
	}

	child := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "child-catalog"},
		Extends:  []gemara.ArtifactMapping{{ReferenceId: "base-catalog"}},
		Controls: []gemara.Control{
			{Id: "CHILD-001", Title: "Child Control"},
		},
	}

	policy := &gemara.Policy{
		Metadata: gemara.Metadata{
			Id: "test-policy",
			MappingReferences: []gemara.MappingReference{
				{Id: "child-catalog"},
			},
		},
		Imports: gemara.Imports{
			Catalogs: []gemara.CatalogImport{
				{ReferenceId: "child-catalog"},
			},
		},
	}

	set := &ArtifactSet{
		Catalogs: map[string]*gemara.ControlCatalog{
			"base-catalog":  base,
			"child-catalog": child,
		},
		Policies: map[string]*gemara.Policy{"test-policy": policy},
		Guidance: make(map[string]*gemara.GuidanceCatalog),
	}

	rp, err := ResolvePolicy(*policy, set)
	require.NoError(t, err)
	assert.Len(t, rp.ControlCatalogs[0].Controls, 2)
}

func TestResolvePolicy_WithCatalogImports(t *testing.T) {
	shared := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{Id: "shared-controls"},
		Controls: []gemara.Control{
			{Id: "SHARED-001", Title: "Shared Control 1"},
			{Id: "SHARED-002", Title: "Shared Control 2"},
			{Id: "SHARED-003", Title: "Shared Control 3"},
		},
	}

	t.Run("selective import includes only specified entries", func(t *testing.T) {
		importing := &gemara.ControlCatalog{
			Metadata: gemara.Metadata{Id: "importing-catalog"},
			Imports: []gemara.MultiEntryMapping{
				{
					ReferenceId: "shared-controls",
					Entries: []gemara.ArtifactMapping{
						{ReferenceId: "SHARED-001"},
						{ReferenceId: "SHARED-003"},
					},
				},
			},
			Controls: []gemara.Control{
				{Id: "LOCAL-001", Title: "Local Control"},
			},
		}

		policy := &gemara.Policy{
			Metadata: gemara.Metadata{
				Id:                "test-policy",
				MappingReferences: []gemara.MappingReference{{Id: "importing-catalog"}},
			},
			Imports: gemara.Imports{
				Catalogs: []gemara.CatalogImport{{ReferenceId: "importing-catalog"}},
			},
		}

		set := &ArtifactSet{
			Catalogs: map[string]*gemara.ControlCatalog{
				"shared-controls":   shared,
				"importing-catalog": importing,
			},
			Policies: map[string]*gemara.Policy{"test-policy": policy},
			Guidance: make(map[string]*gemara.GuidanceCatalog),
		}

		rp, err := ResolvePolicy(*policy, set)
		require.NoError(t, err)

		controls := rp.ControlCatalogs[0].Controls
		assert.Len(t, controls, 3) // LOCAL-001 + SHARED-001 + SHARED-003
		ids := make([]string, len(controls))
		for i, c := range controls {
			ids[i] = c.Id
		}
		assert.Contains(t, ids, "LOCAL-001")
		assert.Contains(t, ids, "SHARED-001")
		assert.Contains(t, ids, "SHARED-003")
		assert.NotContains(t, ids, "SHARED-002")
	})

	t.Run("empty entries imports all controls", func(t *testing.T) {
		importing := &gemara.ControlCatalog{
			Metadata: gemara.Metadata{Id: "importing-catalog"},
			Imports: []gemara.MultiEntryMapping{
				{ReferenceId: "shared-controls"},
			},
			Controls: []gemara.Control{
				{Id: "LOCAL-001", Title: "Local Control"},
			},
		}

		policy := &gemara.Policy{
			Metadata: gemara.Metadata{
				Id:                "test-policy",
				MappingReferences: []gemara.MappingReference{{Id: "importing-catalog"}},
			},
			Imports: gemara.Imports{
				Catalogs: []gemara.CatalogImport{{ReferenceId: "importing-catalog"}},
			},
		}

		set := &ArtifactSet{
			Catalogs: map[string]*gemara.ControlCatalog{
				"shared-controls":   shared,
				"importing-catalog": importing,
			},
			Policies: map[string]*gemara.Policy{"test-policy": policy},
			Guidance: make(map[string]*gemara.GuidanceCatalog),
		}

		rp, err := ResolvePolicy(*policy, set)
		require.NoError(t, err)
		assert.Len(t, rp.ControlCatalogs[0].Controls, 4) // LOCAL-001 + all 3 shared
	})

	t.Run("missing import source tracked as unresolved", func(t *testing.T) {
		importing := &gemara.ControlCatalog{
			Metadata: gemara.Metadata{Id: "importing-catalog"},
			Imports: []gemara.MultiEntryMapping{
				{ReferenceId: "nonexistent"},
			},
			Controls: []gemara.Control{
				{Id: "LOCAL-001", Title: "Local Control"},
			},
		}

		policy := &gemara.Policy{
			Metadata: gemara.Metadata{
				Id:                "test-policy",
				MappingReferences: []gemara.MappingReference{{Id: "importing-catalog"}},
			},
			Imports: gemara.Imports{
				Catalogs: []gemara.CatalogImport{{ReferenceId: "importing-catalog"}},
			},
		}

		set := &ArtifactSet{
			Catalogs: map[string]*gemara.ControlCatalog{
				"importing-catalog": importing,
			},
			Policies: map[string]*gemara.Policy{"test-policy": policy},
			Guidance: make(map[string]*gemara.GuidanceCatalog),
		}

		rp, err := ResolvePolicy(*policy, set)
		require.NoError(t, err)
		assert.Contains(t, rp.Unresolved, "nonexistent")
		assert.Len(t, rp.ControlCatalogs[0].Controls, 1)
	})
}
