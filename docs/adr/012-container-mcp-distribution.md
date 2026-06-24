# ADR 012: Container-Based MCP Server Distribution

**Status:** Proposed

**Date:** 2026-06-08

**Context:**

ComplyPack's MCP server is a Go binary that users must build locally. Issue #24 requires single-click distribution. Four options were evaluated:

1. **Container image** — `docker run --rm -i ghcr.io/complytime/complypack`
2. **Binary download via plugin SessionStart hook** — download pre-built binaries from GitHub Releases into `~/.claude/plugins/data/`
3. **`go install`** — users install via Go toolchain
4. **Homebrew tap** — formula for macOS/Linux

Option 2 was rejected on supply chain grounds: no plugin in the Claude Code ecosystem ships unsigned binaries, and downloading executables into the plugin data directory has no verification standard. The MCP security surface is already a known concern (CVE-2025-59536, CVE-2026-21852).

Option 3 requires the Go toolchain on every user's machine — a non-starter for non-developer users.

Option 4 adds a distribution channel to maintain and doesn't cover Fedora users natively.

**Decision:**

Distribute the MCP server as a multi-arch container image (`ghcr.io/complytime/complypack`). Sign images with cosign (keyless/OIDC). Users invoke via `docker run --rm -i` or `podman run --rm -i` in their `.mcp.json`.

**Consequences:**

- Users must have Docker or Podman installed — acceptable for the target audience (macOS and Fedora developers)
- Container startup adds ~1-2s latency on first invocation (image pull is one-time)
- OCI registry authentication for pulling Gemara catalogs works from inside the container (standard Docker credential chain)
- Image size will be ~30-50MB (Go static binary in distroless/alpine)
- No unsigned binary downloads, no Go toolchain requirement
