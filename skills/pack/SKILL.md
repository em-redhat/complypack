---
name: pack
description: Use when user wants to generate Rego policies from Gemara catalogs, extract assessment requirements and parameters, or work with compliance validation for Kubernetes, Terraform, Docker, Ansible, or CI platforms
---

# /comply:pack — Rego Policy Generation and Assessment

Generate Rego policies from Gemara Control Catalogs that enforce compliance requirements. Policies must be written to disk, validated against the target platform schema, and tested with sample inputs.

**Core principle:** Read control definitions from source → Generate platform-specific policy → Write to disk → Verify it works.

## When to Use

- User requests "generate policy for control X"
- User specifies a Gemara catalog and target platform
- User mentions Conftest, OPA, or Rego
- Generating compliance policies from security frameworks

Do NOT use for:
- Writing arbitrary Rego policies (not from Gemara controls)
- Generating policies without a source catalog

## Quick Reference

| Step | Action | Output |
| ---- | ------ | ------ |
| 1. Read control | Get definition from catalog (MCP) | Control text, ID, title |
| 2. Get parameters | Extract assessment requirements | Thresholds, values |
| 3. Read schema | Get platform schema (MCP) | JSON Schema or CUE |
| 4. Choose format | OPA (allow) or Conftest (deny) | Policy structure |
| 5. Generate policy | Write Rego with control mapping | .rego file |
| 6. Write to disk | Save to `policy/` | File on disk |
| 7. Verify | Test with sample input | Pass/fail results |

## Safety

**DO NOT generate from general knowledge.** Always read the actual control text from MCP.

## MCP Resources and Tools

- `complypack://catalog/*` — Control Catalogs, Guidance Catalogs, Policies
- `complypack://schema/*` — Platform schemas
- `complypack://evaluator` — Available evaluators
- `get_assessment_requirements` — Extract assessment requirements with parameters
- `validate_policy` — Validate policy syntax and contract compliance
- `test_policy` — Run policy tests against sample data
