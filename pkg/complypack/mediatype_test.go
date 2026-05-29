// SPDX-License-Identifier: Apache-2.0

package complypack_test

import (
	"testing"

	"github.com/complytime/complypack/pkg/complypack"
)

func TestMediaTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "artifact type",
			constant: complypack.MediaTypeArtifact,
			expected: "application/vnd.complypack.artifact.v1",
		},
		{
			name:     "config layer",
			constant: complypack.MediaTypeConfig,
			expected: "application/vnd.complypack.config.v1+json",
		},
		{
			name:     "content layer",
			constant: complypack.MediaTypeContent,
			expected: "application/vnd.complypack.content.v1.tar+gzip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("got %q, want %q", tt.constant, tt.expected)
			}
		})
	}
}
