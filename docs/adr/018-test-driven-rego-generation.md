# ADR 018: Test-Driven Rego Policy Generation

**Status:** Accepted

**Date:** 2026-07-08

**Context:**

`validate_policy` and `test_policy` catch structural errors (syntax, wrong `input.*` paths) but not semantic errors — a policy that enforces the wrong condition entirely passes all current gates. The problem compounds with weaker models that skip MCP calls and generate from training data.

Three approaches were considered:

1. **Post-hoc validation only** (status quo) — catches structural errors, not semantic ones.
2. **Human-written test cases** — semantically correct, but requires Rego knowledge the pipeline is meant to replace.
3. **Test-first generation** — generate human-reviewable test cases from MCP data, get approval, then generate policy to satisfy them.

Option 3 was chosen: test cases are reviewable by practitioners in domain terms without Rego expertise, and hallucinated parameters are detectable during review.

**Decision:**

Generate test cases before policy. The user approves test scenarios as the semantic specification; policy is then generated to satisfy them. If tests fail, the policy is revised — never the approved tests.

`comply:build-assessment` is the single user-facing entry point, supporting single mode (one requirement at a time, default) and batch mode. It orchestrates two internal skills (`test-driven-assessment` for test generation, `pack-assessment` for policy generation), both marked `invocable: false`.

**Consequences:**

- Human review moves to the test case layer — practitioners evaluate scenarios in domain terms, not Rego syntax
- Semantic incorrectness is caught before policy is written, not after
- The test suite is a durable artifact alongside the policy
- Generation requires an additional human interaction before policy generation begins
