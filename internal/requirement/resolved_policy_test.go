// SPDX-License-Identifier: Apache-2.0

package requirement

import (
	"testing"

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
