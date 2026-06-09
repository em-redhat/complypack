// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"cuelang.org/go/cue"
)

// URLLoader loads schemas from HTTP/HTTPS URLs.
type URLLoader struct{}

func (l *URLLoader) Match(source string) bool {
	return strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
}

func (l *URLLoader) Load(ctx context.Context, source string, platform string) (*Schema, error) {
	data, format, err := fetchSchemaFromURL(ctx, source)
	if err != nil {
		return nil, err
	}
	if format != FormatJSON {
		return nil, fmt.Errorf("expected JSON format, got %v", format)
	}

	return &Schema{
		Platform: platform,
		Bytes:    data,
		CUE:      cue.Value{},
	}, nil
}

func fetchSchemaFromURL(ctx context.Context, url string) ([]byte, SchemaFormat, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, FormatUnknown, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, FormatUnknown, fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, FormatUnknown, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, FormatUnknown, fmt.Errorf("reading response: %w", err)
	}

	format := DetectFormat(url)
	return data, format, nil
}
