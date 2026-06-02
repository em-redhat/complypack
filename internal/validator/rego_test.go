// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckRego(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantErrs bool
	}{
		{
			name: "valid Rego v1 policy",
			src: `package example
import rego.v1

allow if {
	input.user == "admin"
}`,
			wantErrs: false,
		},
		{
			name: "syntax error",
			src: `package example
allow {
	input.user == "admin"
`,
			wantErrs: true,
		},
		{
			name: "compilation error - undefined function",
			src: `package example
import rego.v1

allow if {
	undefined_function(input.user)
}`,
			wantErrs: true,
		},
		{
			name:     "empty source",
			src:      "",
			wantErrs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := CheckRego("test.rego", tt.src)
			if tt.wantErrs {
				assert.NotEmpty(t, errs, "expected validation errors")
			} else {
				require.Empty(t, errs, "expected no validation errors")
			}
		})
	}
}

func TestCheckRegoMultipleErrors(t *testing.T) {
	src := `package example

# Missing import rego.v1
allow {
	input.user == "admin"
	undefined_func()
}

# Another undefined function
deny {
	another_undefined()
}`

	errs := CheckRego("test.rego", src)
	assert.NotEmpty(t, errs, "should return errors for malformed Rego")
}
