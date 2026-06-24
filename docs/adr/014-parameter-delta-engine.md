# ADR 014: Parameter Delta Gathering Engine

**Status:** Proposed

**Date:** 2026-06-10

**Context:**

Gemara Policies bind parameters at the org level as structured YAML (L3), while Guidance Catalogs (L1) and Control Catalogs (L2) express parameter expectations in prose — requirement text like "the system MUST require multi-factor authentication" or "builds SHOULD achieve at least SLSA Build Level 1."

When preparing the mapping stage of the comply pipeline, the model needs to see L3 parameter values alongside the L1/L2 requirement text they map to. Without tooling, the model must make many MCP calls and manually cross-reference requirement IDs to build this picture.

Three approaches were considered:

1. **No tool** — the model reads policy and catalog resources via MCP and cross-references manually. Works but requires many calls and is error-prone for large catalogs.
2. **Heuristic comparison engine** — classify parameter specificity (concrete vs generic) via string matching, compute verdicts (aligned, mismatch, org_binds_generic). Rejected: heuristics for detecting generic language are brittle, and interpreting whether values differ meaningfully is what the model does well.
3. **Gathering engine** — walk the resolved policy graph, pair each structured L3 parameter with the L1/L2 requirement text it maps to, return them side by side. Let the model interpret the relationship.

Option 3 was chosen. The engine handles what it's good at — traversing the resolved policy graph and collecting structured pairs. The model handles what it's good at — interpreting prose and judging parameter relationships.

**Decision:**

Implement a parameter gathering engine (`requirement.AnalyzeDelta`) that pairs L3 parameter values with L1/L2 requirement text across the resolved policy graph. Each pair contains:

- `requirement_id` — which requirement the parameter maps to
- `label` — the parameter name
- `policy_value` — the structured value from the L3 Policy
- `requirement_text` — the prose from the L1/L2 catalog

Expose this as the `analyze_parameter_delta` MCP tool so the `/comply` pipeline's mapping stage can read comparisons directly from the server. The mapping stage model interprets each pair in domain context and presents its assessment to the user.

**Consequences:**

- The mapping stage consumes structured pairs from MCP rather than manually cross-referencing artifacts
- Interpretation of parameter relationships (which is stricter, whether values conflict) is the model's responsibility — no heuristics
- The engine is intentionally simple: traverse the graph, collect pairs, return them
