// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"context"
	"fmt"

	"cuelang.org/go/cue/cuecontext"
	"github.com/complytime/complypack/schemas"
)

// EmbeddedLoader loads schemas from built-in embedded files.
type EmbeddedLoader struct{}

func (l *EmbeddedLoader) Match(source string) bool {
	return source == ""
}

func (l *EmbeddedLoader) Load(ctx context.Context, source string, platform string) (*Schema, error) {
	data, err := schemas.GetBuiltInSchema(platform)
	if err != nil {
		return nil, fmt.Errorf("no embedded schema for %s: %w", platform, err)
	}

	schemaBytes, err := schemas.GetBuiltInCUESchema(platform)
	if err != nil {
		return nil, fmt.Errorf("failed to load CUE schema for %s: %w", platform, err)
	}

	cueCtx := cuecontext.New()
	cueVal := cueCtx.CompileBytes(schemaBytes)
	if cueVal.Err() != nil {
		return nil, fmt.Errorf("failed to compile CUE schema for %s: %w", platform, cueVal.Err())
	}

	return &Schema{
		Platform: platform,
		Bytes:    data,
		CUE:      cueVal,
	}, nil
}
