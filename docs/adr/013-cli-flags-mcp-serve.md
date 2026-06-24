# ADR 013: CLI Flags for `mcp serve` Configuration

**Status:** Proposed

**Date:** 2026-06-08

**Context:**

`mcp serve` requires a `complypack.yaml` config file. In a containerized deployment, getting a config file into the container requires a volume mount (`-v ./complypack.yaml:/config/complypack.yaml:ro`), which adds friction — the file must exist at a known path before the MCP server starts.

The MCP server only uses two config sections: `gemara.sources` (which OCI artifacts to load) and `schemas` (which platform schemas to serve). Fields like `id`, `evaluator-id`, `policies.dir`, and `output.dir` are only used by `pack` and `scan`.

**Decision:**

Add repeatable `--source` and `--schema` flags to `mcp serve`:

```shell
complypack mcp serve \
  --source oci://registry.example.com/gemara/controls:v1 \
  --source oci+http://localhost:5001/gemara/guidance:v1 \
  --schema ci=cue://cue.dev/x/githubactions@v0#Workflow \
  --schema kubernetes
```

**`--source`** accepts `oci://` (TLS) or `oci+http://` (plain HTTP) URIs. The `+http` scheme variant provides per-source plain-HTTP control without a global flag.

**`--schema`** accepts either a bare platform name (`kubernetes` — uses embedded schema) or `platform=source` syntax (`ci=cue://cue.dev/x/githubactions@v0#Workflow` — loads from the specified source).

When `--source` flags are present, they replace the config file for source resolution. `--config` remains supported and takes precedence if both are provided.

**Consequences:**

- Containerized MCP servers need no volume mount — all configuration passes through `args` in `.mcp.json`
- Users edit one file (`.mcp.json`) instead of two (`.mcp.json` + `complypack.yaml`)
- `pack` and `scan` commands are unaffected — they continue to require `complypack.yaml`
- The `oci+http://` URI scheme is non-standard but self-documenting and avoids a global `--plain-http` flag that would apply to all sources
