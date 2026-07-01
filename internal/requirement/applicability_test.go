// SPDX-License-Identifier: Apache-2.0

package requirement

import (
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
)

func testApplicabilityResolvedPolicy() *ResolvedPolicy {
	catalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id: "app-catalog",
			ApplicabilityGroups: []gemara.Group{
				{Id: "kubernetes", Title: "Kubernetes Workloads", Description: "Controls for Kubernetes clusters"},
				{Id: "docker", Title: "Docker Containers", Description: "Controls for standalone Docker deployments"},
			},
		},
		Controls: []gemara.Control{
			{
				Id: "CTL-001",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTL-001-AR1", Text: "Both platforms", Applicability: []string{"kubernetes", "docker"}},
					{Id: "CTL-001-AR2", Text: "Kubernetes only", Applicability: []string{"kubernetes"}},
				},
			},
			{
				Id: "CTL-002",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTL-002-AR1", Text: "Docker only", Applicability: []string{"docker"}},
					{Id: "CTL-002-AR2", Text: "No group"},
				},
			},
		},
	}

	policy := gemara.Policy{
		Metadata: gemara.Metadata{
			Id:                "app-policy",
			MappingReferences: []gemara.MappingReference{{Id: "app-catalog"}},
		},
		Imports: gemara.Imports{
			Catalogs: []gemara.CatalogImport{{ReferenceId: "app-catalog"}},
		},
	}

	set := &ArtifactSet{
		Catalogs: map[string]*gemara.ControlCatalog{"app-catalog": catalog},
		Policies: map[string]*gemara.Policy{"app-policy": &policy},
		Guidance: make(map[string]*gemara.GuidanceCatalog),
	}

	rp, err := ResolvePolicy(policy, set)
	if err != nil {
		panic(err)
	}
	return rp
}

func TestCollectApplicabilityGroups(t *testing.T) {
	rp := testApplicabilityResolvedPolicy()

	t.Run("all requirements", func(t *testing.T) {
		result := CollectApplicabilityGroups(rp, nil)
		assert.Len(t, result.Groups, 2)
		assert.Len(t, result.Ungrouped, 1)
		assert.Contains(t, result.Ungrouped, "CTL-002-AR2")

		groupByID := make(map[string]ApplicabilityGroupInfo)
		for _, g := range result.Groups {
			groupByID[g.ID] = g
		}

		k8s := groupByID["kubernetes"]
		assert.Equal(t, "Kubernetes Workloads", k8s.Title)
		assert.Contains(t, k8s.RequirementIDs, "CTL-001-AR1")
		assert.Contains(t, k8s.RequirementIDs, "CTL-001-AR2")
		assert.Len(t, k8s.RequirementIDs, 2)

		docker := groupByID["docker"]
		assert.Equal(t, "Docker Containers", docker.Title)
		assert.Contains(t, docker.RequirementIDs, "CTL-001-AR1")
		assert.Contains(t, docker.RequirementIDs, "CTL-002-AR1")
		assert.Len(t, docker.RequirementIDs, 2)
	})

	t.Run("filtered requirements", func(t *testing.T) {
		result := CollectApplicabilityGroups(rp, []string{"CTL-002-AR1"})
		assert.Len(t, result.Groups, 1)
		assert.Equal(t, "docker", result.Groups[0].ID)
		assert.Equal(t, []string{"CTL-002-AR1"}, result.Groups[0].RequirementIDs)
		assert.Empty(t, result.Ungrouped)
	})

	t.Run("filter to ungrouped only", func(t *testing.T) {
		result := CollectApplicabilityGroups(rp, []string{"CTL-002-AR2"})
		assert.Empty(t, result.Groups)
		assert.Equal(t, []string{"CTL-002-AR2"}, result.Ungrouped)
	})

	t.Run("empty filter returns all", func(t *testing.T) {
		result := CollectApplicabilityGroups(rp, []string{})
		assert.Len(t, result.Groups, 2)
		assert.Len(t, result.Ungrouped, 1)
	})
}
