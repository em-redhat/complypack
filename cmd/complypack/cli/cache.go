// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"log"
	"os"

	"github.com/complytime/complypack/internal/cache"
	"github.com/spf13/cobra"
)

// cacheCmd creates the "cache" parent command.
func cacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the OCI artifact cache",
		Long:  "Commands for managing the persistent OCI artifact cache.",
	}

	cmd.AddCommand(cacheCleanCmd())

	return cmd
}

// cacheCleanCmd creates the "cache clean" command.
func cacheCleanCmd() *cobra.Command {
	var cacheDir string

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove all cached OCI artifacts",
		Long: `Remove all cached OCI artifacts from the cache directory.

The cache directory is resolved in order:
  1. --cache-dir flag value
  2. $XDG_CACHE_HOME/complypack
  3. $HOME/.complypack/cache

Examples:
  complypack cache clean
  complypack cache clean --cache-dir /tmp/my-cache`,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedCacheDir, err := cache.ResolveDir(cacheDir)
			if err != nil {
				return fmt.Errorf("failed to resolve cache directory: %w", err)
			}

			// Check if cache exists
			if _, err := os.Stat(resolvedCacheDir); os.IsNotExist(err) {
				log.Printf("No cache exists at %s", resolvedCacheDir)
				return nil
			}

			if err := cache.Clean(resolvedCacheDir); err != nil {
				return fmt.Errorf("failed to clean cache: %w", err)
			}

			log.Printf("Cleaned cache at %s", resolvedCacheDir)
			return nil
		},
	}

	cmd.Flags().StringVar(&cacheDir, "cache-dir", "", "Cache directory (default: $XDG_CACHE_HOME/complypack or $HOME/.complypack/cache)")

	return cmd
}
