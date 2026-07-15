# ADR 018: Lola Module for Skill Distribution

**Status:** Proposed

**Date:** 2026-07-13

**Context:**

Complypack provides AI coding assistants with compliance pipeline skills (audit-pipeline, mcp-setup, pack-assessment) and slash commands (comply-setup, comply-pipeline, comply-pack). These artifacts are authored in two canonical locations:

- `skills/` — skill definitions (markdown with frontmatter)
- `.opencode/commands/` — slash command definitions

Consumer repositories need these skills and commands installed into their own tool-specific directories. Before this change, the only installation path was manual: clone the repo, copy files into the right directories, and repeat on every update. Tool-specific plugin directories (`.claude-plugin/`, `.cursor-plugin/`, `gemini-extension.json`) reference `skills/` directly but only work within the complypack repository itself.

[Lola](https://lobstertrap.org/lola/concepts/skills-and-modules/) provides a standard layout (`module/`) and install command (`lola install`) for distributing AI coding assistant artifacts across repositories. It currently supports OpenCode as the install target.

The challenge is maintaining two sets of files — canonical sources and distributable copies — without them silently drifting apart.

**Decision:**

Add a `module/` directory following the lola AI Context Module layout. Module files are exact copies of their canonical sources, not symlinks, because lola requires a self-contained directory that it can copy into consumer repositories.

**Drift prevention** uses two Taskfile targets:

1. **`task sync-module`** — copies canonical sources into `module/`. Run by contributors after editing canonical files.
2. **`task check-module-sync`** — diffs module copies against sources, exits non-zero on divergence. A `Module Sync Check` job in `ci_local.yml` runs this on every push and PR, preventing drift from reaching `main`.

A pre-commit hook to automate step 1 is tracked separately.

**`mcps.json` version strategy:** The module includes a `mcps.json` with `VERSION` placeholders rather than a pinned release tag or a floating tag like `latest`/`main`. Floating tags were rejected per the constitution's container standards ("MUST use a specific base image tag or digest. MUST NOT use `latest` or floating tags"). Pinning a release tag would go stale on next release. The `mcp-setup` skill handles interactive version resolution at runtime, so the placeholder is the honest representation — users run `/comply-setup` or substitute the version manually.

**Command scope:** Only the three `/comply-*` commands (comply-setup, comply-pipeline, comply-pack) are included. The `review-pr.md` command (a general-purpose PR review workflow) is excluded because it is not a compliance command and consumer repos may have their own review workflows.

**Module `AGENTS.md` scope:** The module's `AGENTS.md` is a consumer-facing overview — what complypack is, what the skills do, the pipeline flow, and how to invoke commands. It does not reuse the root `AGENTS.md`, which contains contributor-focused architecture notes (domain packages, transport layers, testing split) that are irrelevant to consumers.

The existing tool-specific integrations (`.claude-plugin/`, `.cursor-plugin/`, `.opencode/skills/` symlinks, `gemini-extension.json`) remain unchanged. The `.opencode/skills/` symlinks serve local complypack development — they point back to `skills/` at the repo root so OpenCode discovers the skills when working on this repo. They are not part of any distribution mechanism.

**Consequences:**

**Benefits:**

- Consumer repos install with `lola mod add` + `lola install` instead of manual file copying
- CI catches drift between canonical sources and module copies before merge
- Existing tool integrations are unaffected (backwards compatible)
- Canonical sources remain the single source of truth

**Drawbacks:**

- Every canonical file has an exact copy in `module/`, increasing maintenance surface
- Contributors must run `task sync-module` after editing canonical files (until pre-commit hook is added)
- Lola currently supports OpenCode only; other tools still require their own plugin directories

**Related:**

- ADR 012: Container-Based MCP Server Distribution
- ADR 015: Comply Pipeline as Plugin Skills
