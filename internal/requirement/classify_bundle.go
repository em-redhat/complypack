// SPDX-License-Identifier: Apache-2.0

package requirement

import (
	"fmt"

	"github.com/gemaraproj/go-gemara/bundle"
)

// ClassifyBundle classifies a bundle's source and imports into an ArtifactSet.
func ClassifyBundle(b *bundle.Bundle) (*ArtifactSet, error) {
	if len(b.Source.Data) == 0 {
		return nil, fmt.Errorf("bundle has no primary files")
	}

	result, err := Classify(b.Source.Data)
	if err != nil {
		return nil, fmt.Errorf("classifying leaf artifacts: %w", err)
	}

	if len(result.Policies) > 1 {
		return nil, fmt.Errorf("bundle contains %d policy artifacts, expected at most 1", len(result.Policies))
	}
	if len(result.Catalogs) > 1 {
		return nil, fmt.Errorf("bundle contains %d control catalog artifacts, expected at most 1", len(result.Catalogs))
	}

	var depData [][]byte
	for _, f := range b.Imports {
		depData = append(depData, f.Data)
	}

	if len(depData) > 0 {
		imports, err := Classify(depData...)
		if err != nil {
			return nil, fmt.Errorf("classifying imports: %w", err)
		}
		if err := result.Merge(imports); err != nil {
			return nil, fmt.Errorf("merging imports: %w", err)
		}
	}

	return result, nil
}
