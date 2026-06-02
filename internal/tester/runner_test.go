// SPDX-License-Identifier: Apache-2.0

package tester

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		files      map[string]string
		wantTotal  int
		wantPassed int
		wantFailed int
		wantErr    bool
	}{
		{
			name: "all tests pass",
			files: map[string]string{
				"policy_test.rego": `package example
import rego.v1

test_allow if {
	allow with input as {"user": "admin"}
}

allow if {
	input.user == "admin"
}`,
			},
			wantTotal:  1,
			wantPassed: 1,
			wantFailed: 0,
			wantErr:    false,
		},
		{
			name: "one test fails",
			files: map[string]string{
				"policy_test.rego": `package example
import rego.v1

test_pass if {
	allow with input as {"user": "admin"}
}

test_fail if {
	# This test expects allow to be true for guest, but it won't be
	allow with input as {"user": "guest"}
}

allow if {
	input.user == "admin"
}`,
			},
			wantTotal:  2,
			wantPassed: 1,
			wantFailed: 1,
			wantErr:    false,
		},
		{
			name: "parse error",
			files: map[string]string{
				"bad.rego": `package example
allow {  # Missing import rego.v1
	input.user == "admin"
`,
			},
			wantErr: true,
		},
		{
			name:       "empty files map",
			files:      map[string]string{},
			wantTotal:  0,
			wantPassed: 0,
			wantFailed: 0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Run(ctx, tt.files)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, results.Total, "total tests")
			assert.Equal(t, tt.wantPassed, results.Passed, "passed tests")
			assert.Equal(t, tt.wantFailed, results.Failed, "failed tests")

			if tt.wantFailed > 0 {
				assert.NotEmpty(t, results.Errors, "should have error messages for failures")
			}
		})
	}
}
