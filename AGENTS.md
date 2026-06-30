# Agent Instructions

## Architecture

### Thin transport layers

MCP handlers (`internal/mcp/`) and CLI commands (`cmd/complypack/cli/`) are thin wiring: parse input, call a domain function, serialize output. No business logic in these layers.

Business logic belongs in domain packages:
- `internal/requirement/` — policy resolution, triage, delta analysis
- `internal/prepack/` — contract validation
- `internal/evaluator/` — policy evaluation
- `internal/schema/` — schema loading and registry

When adding a new MCP tool or CLI command, write the logic as an exported function in the appropriate domain package first, then wire it from the transport layer.

### Testing follows the same split

Domain package tests cover logic and edge cases. Transport layer tests only verify wiring: correct input parsing, delegation to the domain function, and response serialization.
