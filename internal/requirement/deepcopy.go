// SPDX-License-Identifier: Apache-2.0

package requirement

import "github.com/gemaraproj/go-gemara"

func deepCopyControls(src []gemara.Control) []gemara.Control {
	if src == nil {
		return nil
	}
	dst := make([]gemara.Control, len(src))
	copy(dst, src)
	for i, c := range dst {
		if c.AssessmentRequirements != nil {
			ars := make([]gemara.AssessmentRequirement, len(c.AssessmentRequirements))
			for j, ar := range c.AssessmentRequirements {
				ar.Applicability = copyStrings(ar.Applicability)
				ar.ReplacedBy = cloneEntryMapping(ar.ReplacedBy)
				ars[j] = ar
			}
			dst[i].AssessmentRequirements = ars
		}
		dst[i].Guidelines = copyMultiEntryMappings(c.Guidelines)
		dst[i].Threats = copyMultiEntryMappings(c.Threats)
		dst[i].ReplacedBy = cloneEntryMapping(c.ReplacedBy)
	}
	return dst
}

func deepCopyGuidelines(src []gemara.Guideline) []gemara.Guideline {
	if src == nil {
		return nil
	}
	dst := make([]gemara.Guideline, len(src))
	copy(dst, src)
	for i, g := range dst {
		dst[i].Recommendations = copyStrings(g.Recommendations)
		dst[i].Applicability = copyStrings(g.Applicability)
		dst[i].Statements = copyStatements(g.Statements)
		dst[i].Principles = copyMultiEntryMappings(g.Principles)
		dst[i].Vectors = copyMultiEntryMappings(g.Vectors)
		dst[i].SeeAlso = copyStrings(g.SeeAlso)
		dst[i].Extends = cloneEntryMapping(g.Extends)
		dst[i].ReplacedBy = cloneEntryMapping(g.ReplacedBy)
		dst[i].Rationale = cloneRationale(g.Rationale)
	}
	return dst
}

func deepCopyMetadata(src gemara.Metadata) gemara.Metadata {
	src.MappingReferences = copyMappingReferences(src.MappingReferences)
	src.ApplicabilityGroups = deepCopyGroups(src.ApplicabilityGroups)
	if src.Lexicon != nil {
		cp := *src.Lexicon
		src.Lexicon = &cp
	}
	return src
}

func deepCopyGroups(src []gemara.Group) []gemara.Group {
	if src == nil {
		return nil
	}
	dst := make([]gemara.Group, len(src))
	copy(dst, src)
	return dst
}

func deepCopyExemptions(src []gemara.Exemption) []gemara.Exemption {
	if src == nil {
		return nil
	}
	dst := make([]gemara.Exemption, len(src))
	copy(dst, src)
	for i, e := range dst {
		if e.Redirect != nil {
			cp := *e.Redirect
			cp.Entries = make([]gemara.ArtifactMapping, len(e.Redirect.Entries))
			copy(cp.Entries, e.Redirect.Entries)
			dst[i].Redirect = &cp
		}
	}
	return dst
}

func copyStrings(src []string) []string {
	if src == nil {
		return nil
	}
	dst := make([]string, len(src))
	copy(dst, src)
	return dst
}

func copyMultiEntryMappings(src []gemara.MultiEntryMapping) []gemara.MultiEntryMapping {
	if src == nil {
		return nil
	}
	dst := make([]gemara.MultiEntryMapping, len(src))
	for i, m := range src {
		dst[i] = m
		if m.Entries != nil {
			entries := make([]gemara.ArtifactMapping, len(m.Entries))
			copy(entries, m.Entries)
			dst[i].Entries = entries
		}
	}
	return dst
}

func copyMappingReferences(src []gemara.MappingReference) []gemara.MappingReference {
	if src == nil {
		return nil
	}
	dst := make([]gemara.MappingReference, len(src))
	copy(dst, src)
	return dst
}

func copyStatements(src []gemara.Statement) []gemara.Statement {
	if src == nil {
		return nil
	}
	dst := make([]gemara.Statement, len(src))
	for i, s := range src {
		dst[i] = s
		dst[i].Recommendations = copyStrings(s.Recommendations)
	}
	return dst
}

func cloneEntryMapping(src *gemara.EntryMapping) *gemara.EntryMapping {
	if src == nil {
		return nil
	}
	cp := *src
	return &cp
}

func cloneRationale(src *gemara.Rationale) *gemara.Rationale {
	if src == nil {
		return nil
	}
	cp := *src
	cp.Goals = copyStrings(src.Goals)
	return &cp
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}
