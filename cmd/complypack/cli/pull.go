// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"log"
	"strings"

	"github.com/complytime/complypack/internal/cache"
	"github.com/complytime/complypack/internal/config"
	"github.com/complytime/complypack/internal/source"
	"github.com/spf13/cobra"
)

// pullCmd creates the "pull" command.
func pullCmd() *cobra.Command {
	var (
		configPath string
		cacheDir   string
	)

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pre-warm the OCI cache for all configured sources",
		Long: `Pull all configured Gemara OCI sources into the local cache.

This command reads complypack.yaml, iterates all gemara sources,
and pulls each OCI source into the persistent cache directory.
File sources (file://) are skipped since they are already local.

The cache directory is resolved in order:
  1. --cache-dir flag value
  2. $XDG_CACHE_HOME/complypack
  3. $HOME/.complypack/cache

Examples:
  complypack pull
  complypack pull --config complypack.yaml
  complypack pull --cache-dir /tmp/my-cache`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Resolve cache directory
			resolvedCacheDir, err := cache.ResolveDir(cacheDir)
			if err != nil {
				return fmt.Errorf("failed to resolve cache directory: %w", err)
			}

			// Load config
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if err := cfg.ValidateForMCP(); err != nil {
				return fmt.Errorf("failed to validate config: %w", err)
			}

			log.Printf("Cache directory: %s", resolvedCacheDir)

			var pulled, skipped int
			var errors []string

			for _, entry := range cfg.Gemara.Sources {
				src := entry.Source

				// Skip file sources — they are already local
				if strings.HasPrefix(src, "file://") || !isOCISource(src) {
					log.Printf("  Skipping %s (local source)", src)
					skipped++
					continue
				}

				log.Printf("  Pulling %s...", src)
				_, err := source.LoadArtifacts(ctx, src, entry.PlainHTTP, resolvedCacheDir)
				if err != nil {
					log.Printf("  ERROR: %s: %v", src, err)
					errors = append(errors, fmt.Sprintf("%s: %v", src, err))
					continue
				}
				log.Printf("  Pulled %s", src)
				pulled++
			}

			log.Printf("Done: %d pulled, %d skipped", pulled, skipped)
			if len(errors) > 0 {
				log.Printf("Errors:")
				for _, e := range errors {
					log.Printf("  %s", e)
				}
				return fmt.Errorf("%d source(s) failed to pull", len(errors))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "complypack.yaml", "Path to complypack.yaml config file")
	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Cache directory (default: $XDG_CACHE_HOME/complypack or $HOME/.complypack/cache)")

	return cmd
}

// isOCISource returns true if the source looks like an OCI reference.
func isOCISource(source string) bool {
	if strings.HasPrefix(source, "file://") {
		return false
	}
	return strings.HasPrefix(source, "oci://") ||
		strings.HasPrefix(source, "oci+http://") ||
		(strings.Contains(source, "/") && (strings.Contains(source, ":") || strings.Contains(source, "//")))
}
