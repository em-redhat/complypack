// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"context"
	"strings"
)

// LegacyLoader handles bare file paths without any scheme prefix.
type LegacyLoader struct{}

func (l *LegacyLoader) Match(source string) bool {
	return source != "" &&
		!strings.HasPrefix(source, "cue://") &&
		!strings.HasPrefix(source, "https://") &&
		!strings.HasPrefix(source, "http://") &&
		!strings.HasPrefix(source, "file://")
}

func (l *LegacyLoader) Load(ctx context.Context, source string, platform string) (*Schema, error) {
	return loadFromFile(source, platform)
}
