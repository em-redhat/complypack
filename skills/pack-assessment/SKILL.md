---
name: pack-assessment
description: Use when user mentions Rego, Conftest, OPA, "generate policy", or "assessment logic" in the context of Gemara catalogs. Generates Rego policies from Gemara Control Catalogs for Kubernetes (Deployments, Pods, DaemonSets, StatefulSets, CronJobs, Jobs, Services, NetworkPolicies, Ingress, RBAC, ConfigMaps, Secrets), CI/CD pipelines (GitHub Actions, GitLab CI, Azure Pipelines), Conftest, or OPA
---

# /comply:pack-assessment — Rego Policy Generation and Assessment

Generate Rego policies from Gemara Control Catalogs that enforce compliance requirements. Policies must be written to disk, validated against the target platform schema, and tested with sample inputs.

**Core principle:** Read control definitions from source → Generate platform-specific policy → Write to disk → Verify it works.

## Quick Reference

| Step | Action | Output |
| ---- | ------ | ------ |
| 1. Scope | Call `get_automation_triage`, filter to automated plans | Requirement list |
| 2. Read control | Get definition from catalog (MCP) | Control text, ID, title |
| 3. Get parameters | Extract assessment requirements | Thresholds, values |
| 4. Read schema | Get platform schema (MCP) | Platform schema |
| 5. Choose format | Select evaluator output convention | Policy structure |
| 6. Generate policy | Write Rego against schema contract | .rego file |
| 7. Write to disk | Save to `policy/` | File on disk |
| 8. Validate | Contract check then test | Pass/fail results |

## Step 1: Scope — Filter to Automated Requirements

Call `get_automation_triage` with the policy name to get the automation split:

1. The tool returns `automated` and `manual` lists with requirement IDs, evaluation methods, and executor details
2. Generate policies only for requirements in the `automated` list
3. List `manual` requirements for the user — these need human review

If no policy is loaded in the MCP server, proceed with all requested requirements.

## Steps 2-5: Read and Prepare

**DO NOT generate from general knowledge.** Always read the actual control text from MCP.

1. **Read control** — get the definition from the catalog via MCP (`complypack://catalog/*`)
2. **Get parameters** — call `get_assessment_requirements` to extract thresholds and accepted values
3. **Read schema** — get the platform schema via MCP (`complypack://schema/*`)
4. **Choose format** — select evaluator output convention (e.g., OPA `deny` rules for Conftest)

## Step 6: Generate Policy — Reusability Rules

Write policies against the platform schema contract, not sample inputs:

- **Write `input.*` paths from the schema.** Read `complypack://schema/*` and use the paths it defines. Do NOT reverse-engineer paths from sample manifests in `targets/`.
- **No hardcoded values from test data.** Do not embed names, image refs, step names, or other values from sample inputs. Use parameter values from `get_assessment_requirements` for thresholds and accepted values.
- **One file per assessment requirement.** Name the file after the requirement (e.g., `run_as_nonroot.rego`).
- **Identify the subject in denial messages.** Use fields from the schema that identify the resource being checked (e.g., `input.metadata.name` for Kubernetes, `input.name` for CI jobs). Do not hardcode expected values in messages.

## Step 7: Write to Disk

Save to `policy/` directory.

## Step 8: Validate — Contract Check First

1. Run `validate_policy` — confirm zero contract violations against the platform schema
2. If contract violations: fix the `input.*` paths to match the schema. The schema is the source of truth, not test data.
3. Run `test_policy` — confirm policy logic works with sample inputs

## Safety

**DO NOT generate from general knowledge.** Always read the actual control text from MCP.

## MCP Resources and Tools

- `complypack://catalog/*` — Control Catalogs, Guidance Catalogs, Policies
- `complypack://schema/*` — Platform schemas
- `complypack://evaluator` — Available evaluators
- `get_automation_triage` — Classify assessment plans as Automated or Manual with executor details
- `get_assessment_requirements` — Extract assessment requirements with parameters
- `validate_policy` — Validate policy syntax and contract compliance
- `test_policy` — Run policy tests against sample data

## Red Flags - STOP AND FIX IF THERE ARE ISSUES

- [ ] Does every `input.*` reference exist in the platform schema?
- [ ] Are there hardcoded values from sample inputs that should be parameters?
- [ ] Did you run `validate_policy` before `test_policy`?
- [ ] Is each `.rego` file scoped to a single assessment requirement?
- [ ] Did you read control text from MCP, not from general knowledge?
- [ ] Did you call `get_automation_triage` to determine which plans are automated?
