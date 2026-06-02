// SPDX-License-Identifier: Apache-2.0

package tester

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunFixtures(t *testing.T) {
	ctx := context.Background()

	t.Run("pass and fail fixtures", func(t *testing.T) {
		// Create temp dirs
		policyDir := t.TempDir()
		fixtureDir := t.TempDir()

		// Write policy
		policyFile := filepath.Join(policyDir, "policy.rego")
		policy := `package kubernetes
import rego.v1

deny_privileged contains msg if {
	input.kind == "Pod"
	some container in input.spec.containers
	container.securityContext.privileged == true
	msg := sprintf("Container %s is privileged", [container.name])
}`
		err := os.WriteFile(policyFile, []byte(policy), 0600)
		require.NoError(t, err)

		// Write passing fixture
		allowFixture := filepath.Join(fixtureDir, "valid_pod.json")
		allowJSON := `{
  "kind": "Pod",
  "spec": {
    "containers": [
      {
        "name": "app",
        "securityContext": {
          "privileged": false
        }
      }
    ]
  }
}`
		err = os.WriteFile(allowFixture, []byte(allowJSON), 0600)
		require.NoError(t, err)

		// Write failing fixture
		denyFixture := filepath.Join(fixtureDir, "invalid_pod.json")
		denyJSON := `{
  "kind": "Pod",
  "spec": {
    "containers": [
      {
        "name": "bad",
        "securityContext": {
          "privileged": true
        }
      }
    ]
  }
}`
		err = os.WriteFile(denyFixture, []byte(denyJSON), 0600)
		require.NoError(t, err)

		// Run fixtures
		results, err := RunFixtures(ctx, fixtureDir, policyDir)
		require.NoError(t, err)

		assert.Equal(t, 2, results.Total)
		assert.Equal(t, 2, results.Passed)
		assert.Equal(t, 0, results.Failed)
	})

	t.Run("mismatch detection", func(t *testing.T) {
		policyDir := t.TempDir()
		fixtureDir := t.TempDir()

		// Write policy
		policyFile := filepath.Join(policyDir, "policy.rego")
		policy := `package example
import rego.v1

deny_always contains "always denied" if {
	true
}`
		err := os.WriteFile(policyFile, []byte(policy), 0600)
		require.NoError(t, err)

		// Write fixture that expects allow but will get deny
		allowFixture := filepath.Join(fixtureDir, "valid_case.json")
		err = os.WriteFile(allowFixture, []byte(`{}`), 0600)
		require.NoError(t, err)

		results, err := RunFixtures(ctx, fixtureDir, policyDir)
		require.NoError(t, err)

		assert.Equal(t, 1, results.Total)
		assert.Equal(t, 0, results.Passed)
		assert.Equal(t, 1, results.Failed)
		assert.Len(t, results.Errors, 1)
		assert.Equal(t, "allow", results.Errors[0].Expected)
		assert.NotEmpty(t, results.Errors[0].Violations)
	})

	t.Run("non-existent directory", func(t *testing.T) {
		_, err := RunFixtures(ctx, "/nonexistent/fixtures", "/nonexistent/policies")
		assert.Error(t, err)
	})

	t.Run("empty directories", func(t *testing.T) {
		policyDir := t.TempDir()
		fixtureDir := t.TempDir()

		results, err := RunFixtures(ctx, fixtureDir, policyDir)
		require.NoError(t, err)
		assert.Equal(t, 0, results.Total)
	})
}

func TestExpectation(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"allow suffix", "test_allow.json", "allow"},
		{"valid suffix", "test_valid.json", "allow"},
		{"valid prefix", "valid_test.json", "allow"},
		{"deny suffix", "test_deny.json", "deny"},
		{"invalid suffix", "test_invalid.json", "deny"},
		{"invalid prefix", "invalid_test.json", "deny"},
		{"no marker", "test.json", "deny"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expectation(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}
