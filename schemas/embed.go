// SPDX-License-Identifier: Apache-2.0

package schemas

import (
	"embed"
	"fmt"
)

// JSONSchemas embeds all generated JSON Schema files.
//
//go:embed json-schema/*.json
var JSONSchemas embed.FS

//go:embed cue/*.cue
var CUESchemas embed.FS

// BuiltInPlatforms lists all platforms with embedded schemas.
var BuiltInPlatforms = []string{
	"kubernetes",
	"terraform",
	"docker",
	"ansible",
	"ci",
}

// GetBuiltInSchema reads the embedded JSON Schema for a platform.
// Returns error if platform is not built-in.
func GetBuiltInSchema(platform string) ([]byte, error) {
	path := fmt.Sprintf("json-schema/%s.json", platform)
	data, err := JSONSchemas.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("schema not found for platform %q: %w", platform, err)
	}
	return data, nil
}

// GetBuiltInCUESchema reads the embedded CUE schema source for a platform.
// Returns error if platform is not built-in.
func GetBuiltInCUESchema(platform string) ([]byte, error) {
	path := fmt.Sprintf("cue/%s.cue", platform)
	data, err := CUESchemas.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("CUE schema not found for platform %q: %w", platform, err)
	}
	return data, nil
}
