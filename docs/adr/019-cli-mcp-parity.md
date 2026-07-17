# ADR 019: CLI and MCP Transport Parity

**Status:** Proposed

**Date:** 2026-07-17

**Context:**

ComplyPack exposes compliance operations through two transport layers: a CLI
(`cmd/complypack/cli/`) and an MCP server (`internal/mcp/`). Both are thin
wiring over domain packages in `internal/`.

Issue #85 proposed removing the MCP server entirely and replacing it with CLI
commands, arguing that the underlying data is static (cached OCI artifacts) and
does not benefit from a session-based protocol. After discussion, issue #87
reframed the goal: keep both transports but ensure the CLI can do everything
the MCP server can, where it makes sense.

The current gap:

| MCP Tool                       | CLI Equivalent              |
|--------------------------------|-----------------------------|
| `validate_policy`              | Partial (embedded in `pack`)|
| `test_policy`                  | Partial (embedded in `pack`)|
| `get_assessment_requirements`  | None                        |
| `analyze_parameter_delta`      | None                        |
| `get_automation_triage`        | None                        |
| `get_applicability_groups`     | None                        |

Four MCP tools have no CLI equivalent. Two have only partial coverage because
their functionality is embedded inside the `pack` command's pre-pack validation
pipeline rather than exposed as standalone commands.

Additionally, some domain logic currently lives in the MCP transport layer
(`extractFromResolvedPolicy` and `resolveFromCatalog` in
`internal/mcp/tool_assessment.go`) rather than in domain packages, which
prevents the CLI from reusing it.

**Decision:**

Both the CLI and the MCP server are maintained as thin transport layers over
shared domain functions. New capabilities SHOULD be exposed through both
transports where it makes sense:

- **CLI** provides deterministic, scriptable access for automation and CI/CD.
- **MCP** provides conversational, LLM-assisted workflows.

This is a SHOULD-level guideline, not a hard requirement. Some capabilities are
naturally transport-specific (e.g., `complypack init` is interactive and has no
MCP equivalent; MCP resources serve discovery data that a CLI user would not
need). Developers use common sense to decide when parity applies.

To enable parity, shared infrastructure must be extracted from the MCP layer
into domain packages:

1. Domain logic (`extractFromResolvedPolicy`, `resolveFromCatalog`) moves to
   `internal/requirement/`.
2. The artifact loading pipeline (config parsing, source fetching, policy
   resolution) is extracted into a reusable function so both transports can
   load artifacts without duplicating the setup sequence.
3. Schema loading is extracted from `internal/mcp/` into `internal/schema/` so
   both `pack` and new CLI commands share the same path.

**Consequences:**

- Six new CLI commands will be added to close the parity gap: `requirements`,
  `delta`, `triage`, `applicability`, `validate`, and `test`.
- Domain logic must not live in either transport layer. Adding a new capability
  means writing the domain function first, then wiring from both CLI and MCP.
- Both transports support flexible configuration: `--config` for file-based
  setup and `--source`/`--schema` flags for ad-hoc usage (matching ADR 013).
- CLI commands default to human-readable output with `--output json` for
  machine consumption. MCP tools continue to return JSON.
