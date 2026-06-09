// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"context"
	"fmt"
	"os"
	"strings"

	"cuelang.org/go/cue"
)

// FileLoader loads schemas from file:// URIs.
type FileLoader struct{}

func (l *FileLoader) Match(source string) bool {
	return strings.HasPrefix(source, "file://")
}

func (l *FileLoader) Load(ctx context.Context, source string, platform string) (*Schema, error) {
	filePath := strings.TrimPrefix(source, "file://")
	if filePath == "" {
		return nil, fmt.Errorf("file:// scheme requires path")
	}
	return loadFromFile(filePath, platform)
}

func loadFromFile(path string, platform string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	format := DetectFormat(path)

	switch format {
	case FormatJSON:
		return &Schema{
			Platform: platform,
			Bytes:    data,
			CUE:      cue.Value{},
		}, nil
	case FormatCUE:
		cueVal, err := BuildCUEFromBytes(data)
		if err != nil {
			return nil, err
		}
		return &Schema{
			Platform: platform,
			Bytes:    data,
			CUE:      cueVal,
		}, nil
	default:
		return &Schema{
			Platform: platform,
			Bytes:    data,
			CUE:      cue.Value{},
		}, nil
	}
}
