// SPDX-License-Identifier: Apache-2.0

package requirement

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testControlCatalogYAML = `metadata:
  id: test-catalog
  type: ControlCatalog
  gemara-version: "1.0.0"
controls:
  - id: CTRL-001
    title: Test Control
    description: A test control
    assessment-requirements:
      - id: REQ-001
        text: Verify the thing
        applicability:
          - kubernetes
`

const testPolicyYAML = `metadata:
  id: test-policy
  type: Policy
  gemara-version: "1.0.0"
  mapping-references:
    - id: test-catalog
imports:
  catalogs:
    - reference-id: test-catalog
adherence:
  assessment-plans:
    - requirement-id: REQ-001
      parameters:
        - label: timeout
          accepted-values:
            - "30s"
`

const testGuidanceCatalogYAML = `metadata:
  id: test-guidance
  type: GuidanceCatalog
  gemara-version: "1.0.0"
guidance-type: security
guidelines:
  - id: GL-001
    title: Test Guideline
    group: general
`

func TestClassify(t *testing.T) {
	tests := []struct {
		name         string
		data         [][]byte
		wantCatalogs int
		wantPolicies int
		wantGuidance int
		wantErr      bool
	}{
		{
			name:         "control catalog",
			data:         [][]byte{[]byte(testControlCatalogYAML)},
			wantCatalogs: 1,
		},
		{
			name:         "policy",
			data:         [][]byte{[]byte(testPolicyYAML)},
			wantPolicies: 1,
		},
		{
			name:         "guidance catalog",
			data:         [][]byte{[]byte(testGuidanceCatalogYAML)},
			wantGuidance: 1,
		},
		{
			name:         "multiple artifacts",
			data:         [][]byte{[]byte(testControlCatalogYAML), []byte(testPolicyYAML)},
			wantCatalogs: 1,
			wantPolicies: 1,
		},
		{
			name:    "invalid yaml",
			data:    [][]byte{[]byte("invalid: [unclosed")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Classify(tt.data...)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result.Catalogs, tt.wantCatalogs)
			assert.Len(t, result.Policies, tt.wantPolicies)
			assert.Len(t, result.Guidance, tt.wantGuidance)
		})
	}
}

func TestClassify_KeyedByID(t *testing.T) {
	result, err := Classify([]byte(testControlCatalogYAML))
	require.NoError(t, err)

	_, ok := result.Catalogs["test-catalog"]
	assert.True(t, ok, "catalog should be keyed by metadata.id")
}

func TestArtifactSet_Merge(t *testing.T) {
	a := NewArtifactSet()
	b := NewArtifactSet()

	aResult, _ := Classify([]byte(testControlCatalogYAML))
	bResult, _ := Classify([]byte(testPolicyYAML))

	a.Catalogs = aResult.Catalogs
	b.Policies = bResult.Policies

	err := a.Merge(b)
	require.NoError(t, err)
	assert.Len(t, a.Catalogs, 1)
	assert.Len(t, a.Policies, 1)
}

func TestArtifactSet_MergeDuplicateID(t *testing.T) {
	a, _ := Classify([]byte(testControlCatalogYAML))
	b, _ := Classify([]byte(testControlCatalogYAML))

	err := a.Merge(b)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate artifact id")
}
