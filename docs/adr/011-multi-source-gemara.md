# ADR 011: Multi-Source Gemara Artifact Loading

**Status:** Accepted

**Date:** 2026-06-05

**Context:**

The MCP server originally loaded Gemara artifacts from a single source (one file path or OCI reference). Real-world usage requires composing artifacts from multiple independent sources — e.g., an organization's control catalog from one OCI registry and a project-specific policy from a local file. Forcing users to pre-merge artifacts externally adds friction and prevents clean separation of concerns.

**Decision:**

Introduce a `sources` list in the `gemara` config block. Each entry specifies a `source` (file path, `file://`, `oci://`, or bare OCI reference) and optional `plain-http` flag. The server iterates all entries, loads each independently, and merges results into a single `LoadedArtifacts` struct.

Key design choices:

1. **Duplicate-ID detection** — `LoadedArtifacts.Merge` returns an error if any `metadata.id` appears in more than one source. This prevents silent shadowing and forces users to resolve conflicts explicitly.
2. **Independent EffectivePolicy resolution** — each source resolves its own policies against its own catalogs/guidance before merging. Cross-source policy resolution is not supported; a policy must ship with (or bundle) the catalogs it references.
3. **Backward compatibility** — the legacy single-source format (`gemara.source: <path>`) is deserialized into a one-element `Sources` slice via a custom `UnmarshalYAML` hook. Specifying both `source` and `sources` is a hard error.

Config shape:

```yaml
gemara:
  sources:
    - source: catalogs/controls.yaml
    - source: ghcr.io/org/guidance:v1
      plain-http: true
```

**Consequences:**

**Benefits:**

- Users compose catalogs and policies from independent publishers without pre-processing
- Fail-fast on ID collisions prevents ambiguous state at runtime
- Legacy configs continue working with no migration

**Risks:**

- Cross-source policy-to-catalog resolution is intentionally unsupported; users must bundle related artifacts together or use a single source that contains all dependencies
- Ordering in the `sources` list affects error messages but not semantics (merge is commutative aside from error reporting order)
