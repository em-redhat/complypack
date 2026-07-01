// SPDX-License-Identifier: Apache-2.0

package requirement

import (
	"sort"

	"github.com/gemaraproj/go-gemara"
)

// ApplicabilityGroupResult maps applicability group definitions to their member requirements.
type ApplicabilityGroupResult struct {
	Groups    []ApplicabilityGroupInfo `json:"groups"`
	Ungrouped []string                 `json:"ungrouped"`
}

// ApplicabilityGroupInfo describes an applicability group with its member requirements.
type ApplicabilityGroupInfo struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	RequirementIDs []string `json:"requirement_ids"`
}

// CollectApplicabilityGroups builds a group-to-requirements mapping from a
// resolved policy's catalogs. If filterReqIDs is non-empty, only groups
// containing at least one matching requirement are returned.
func CollectApplicabilityGroups(rp *ResolvedPolicy, filterReqIDs []string) *ApplicabilityGroupResult {
	groupDefs := make(map[string]gemara.Group)
	for _, catalog := range rp.ControlCatalogs {
		for _, g := range catalog.Metadata.ApplicabilityGroups {
			if _, exists := groupDefs[g.Id]; !exists {
				groupDefs[g.Id] = g
			}
		}
	}

	filterSet := make(map[string]bool, len(filterReqIDs))
	for _, id := range filterReqIDs {
		filterSet[id] = true
	}

	groupReqs := make(map[string][]string)
	var ungrouped []string

	for _, controlID := range rp.ControlIDs() {
		for _, req := range rp.RequirementsForControl(controlID) {
			if len(filterSet) > 0 && !filterSet[req.Id] {
				continue
			}
			if len(req.Applicability) == 0 {
				ungrouped = append(ungrouped, req.Id)
				continue
			}
			for _, groupID := range req.Applicability {
				groupReqs[groupID] = append(groupReqs[groupID], req.Id)
				if _, exists := groupDefs[groupID]; !exists {
					groupDefs[groupID] = gemara.Group{Id: groupID}
				}
			}
		}
	}

	var groups []ApplicabilityGroupInfo
	for _, g := range groupDefs {
		reqs := groupReqs[g.Id]
		if len(filterSet) > 0 && len(reqs) == 0 {
			continue
		}
		if reqs == nil {
			reqs = []string{}
		}
		groups = append(groups, ApplicabilityGroupInfo{
			ID:             g.Id,
			Title:          g.Title,
			Description:    g.Description,
			RequirementIDs: reqs,
		})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })

	if ungrouped == nil {
		ungrouped = []string{}
	}

	return &ApplicabilityGroupResult{
		Groups:    groups,
		Ungrouped: ungrouped,
	}
}
