// SPDX-License-Identifier: Apache-2.0

package requirement

import (
	"fmt"

	"github.com/gemaraproj/go-gemara"
	goyaml "github.com/goccy/go-yaml"
)

// ArtifactSet holds classified artifacts keyed by metadata.id.
type ArtifactSet struct {
	Catalogs map[string]*gemara.ControlCatalog
	Policies map[string]*gemara.Policy
	Guidance map[string]*gemara.GuidanceCatalog
	Mappings map[string]*gemara.MappingDocument
}

// NewArtifactSet returns an initialized ArtifactSet.
func NewArtifactSet() *ArtifactSet {
	return &ArtifactSet{
		Catalogs: make(map[string]*gemara.ControlCatalog),
		Policies: make(map[string]*gemara.Policy),
		Guidance: make(map[string]*gemara.GuidanceCatalog),
		Mappings: make(map[string]*gemara.MappingDocument),
	}
}

// Classify detects and unmarshals raw artifact data into an ArtifactSet.
func Classify(data ...[]byte) (*ArtifactSet, error) {
	as := NewArtifactSet()
	for i, d := range data {
		artType, err := gemara.DetectType(d)
		if err != nil {
			return nil, fmt.Errorf("artifact %d: %w", i, err)
		}
		switch artType {
		case gemara.PolicyArtifact:
			var p gemara.Policy
			if err := goyaml.Unmarshal(d, &p); err != nil {
				return nil, fmt.Errorf("artifact %d (Policy): %w", i, err)
			}
			as.Policies[p.Metadata.Id] = &p
		case gemara.ControlCatalogArtifact:
			var cc gemara.ControlCatalog
			if err := goyaml.Unmarshal(d, &cc); err != nil {
				return nil, fmt.Errorf("artifact %d (ControlCatalog): %w", i, err)
			}
			as.Catalogs[cc.Metadata.Id] = &cc
		case gemara.GuidanceCatalogArtifact:
			var gc gemara.GuidanceCatalog
			if err := goyaml.Unmarshal(d, &gc); err != nil {
				return nil, fmt.Errorf("artifact %d (GuidanceCatalog): %w", i, err)
			}
			as.Guidance[gc.Metadata.Id] = &gc
		case gemara.MappingDocumentArtifact:
			var md gemara.MappingDocument
			if err := goyaml.Unmarshal(d, &md); err != nil {
				return nil, fmt.Errorf("artifact %d (MappingDocument): %w", i, err)
			}
			as.Mappings[md.Metadata.Id] = &md
		}
	}
	return as, nil
}

// Merge combines another ArtifactSet into this one.
// Returns an error if any artifact ID appears in both sets.
func (as *ArtifactSet) Merge(other *ArtifactSet) error {
	for id, cat := range other.Catalogs {
		if _, exists := as.Catalogs[id]; exists {
			return fmt.Errorf("duplicate artifact id %q across sources", id)
		}
		as.Catalogs[id] = cat
	}
	for id, pol := range other.Policies {
		if _, exists := as.Policies[id]; exists {
			return fmt.Errorf("duplicate artifact id %q across sources", id)
		}
		as.Policies[id] = pol
	}
	for id, gc := range other.Guidance {
		if _, exists := as.Guidance[id]; exists {
			return fmt.Errorf("duplicate artifact id %q across sources", id)
		}
		as.Guidance[id] = gc
	}
	for id, md := range other.Mappings {
		if _, exists := as.Mappings[id]; exists {
			return fmt.Errorf("duplicate artifact id %q across sources", id)
		}
		as.Mappings[id] = md
	}
	return nil
}
