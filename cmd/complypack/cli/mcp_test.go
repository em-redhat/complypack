// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"testing"

	"github.com/complytime/complypack/cmd/complypack/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMcpCommand(t *testing.T) {
	root := cli.New()

	// Find the mcp command
	mcpCmd, _, err := root.Find([]string{"mcp"})
	require.NoError(t, err, "mcp command should exist")
	assert.Equal(t, "mcp", mcpCmd.Name())
	assert.NotEmpty(t, mcpCmd.Short, "mcp command should have a short description")

	// Find the serve subcommand
	serveCmd, _, err := mcpCmd.Find([]string{"serve"})
	require.NoError(t, err, "mcp serve command should exist")
	assert.Equal(t, "serve", serveCmd.Name())
	assert.NotEmpty(t, serveCmd.Short, "serve command should have a short description")

	// Check flags exist
	flags := serveCmd.Flags()
	assert.NotNil(t, flags.Lookup("config"), "should have --config flag")
	assert.NotNil(t, flags.Lookup("cache-dir"), "should have --cache-dir flag")
}
