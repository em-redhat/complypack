---
name: mapping
description: Crosswalk frameworks and perform parameter harmonization. Parent Policy always sets the floor.
user-invocable: false
---

# Mapping — Delta Analysis & Parameter Harmonization

Compare parameters across Guidance Catalogs, the parent Policy, and Control Catalogs. Surface where the organization's values match or differ from each framework. Interpret the relationship using domain context — the tool gathers, the model judges.

**Core invariant:** The parent Policy always sets the floor. Never resolve below it without explicit user acknowledgement.

## Prerequisites

Requires `.complytime/scoping.yaml` from the scoping stage.

## Key Concepts

### Mandated vs. Under Evaluation

The parent Policy's `imports.guidance` determines which frameworks are binding:

- **Mandated:** Guidance Catalogs imported by the parent Policy. Shortfalls MUST be addressed.
- **Under evaluation:** Guidance Catalogs loaded in MCP but NOT imported. Informational only.

## Process

### Step 1: Read Scoping Artifacts

Read `.complytime/scoping.yaml`.

### Step 2: Identify Parent Policy

```text
ListMcpResourcesTool(server="complypack")
```

Look for resources with "Policy" in the name. If a Policy exists, read it. **If no parent Policy exists:** extract minimums from the target framework's Control Catalog requirements. The framework becomes the floor.

### Step 3: Classify Guidance Frameworks

If a parent Policy exists, check `imports.guidance`. If not, all Guidance Catalogs are **under evaluation** unless the user designates one as the target.

### Step 4: Load Mapping Documents

Read any Mapping Documents recorded in scoping:

```text
ReadMcpResourceTool(server="complypack", uri="complypack://mapping/<id>")
```

Use them to resolve framework crosswalks.

### Step 5: Gather Parameter Comparisons

```text
CallMcpToolTool(server="complypack", tool="analyze_parameter_delta", arguments={"policyName": "<policy-name>"})
```

This returns structured L3 parameter values from the Policy alongside the L1/L2 requirement text they map to. Each comparison contains:
- `requirement_id` — which requirement the parameter maps to
- `label` — the parameter name
- `policy_value` — the structured value from the Policy
- `requirement_text` — the prose from the catalog

**The tool does not judge the relationship.** You interpret each pair using domain context.

### Step 6: Present Results

For each comparison, interpret the relationship:
- Do the values align?
- Does the requirement text express a concrete expectation the policy value satisfies?
- Does the policy set a stricter threshold than the requirement implies?
- Is the requirement generic ("per organizational requirements") and the policy provides a concrete binding?

**Mandated frameworks:** present your interpretation and recommended action.

**Under evaluation:** same analysis, framed as "if you pursue this certification..."

### Step 7: Resolve Decisions

Walk the user through items where values differ. For each, present both the policy value and the requirement text, explain your interpretation, and let the user decide.

### Step 8: Write Output

Write `.complytime/delta-report.yaml`:

```yaml
version: "1"
created: YYYY-MM-DD
sources:
  parent_policy: "<policy-id>"
  guidance:
    mandated:
      - id: "<id>"
        status: imported_by_parent
    under_evaluation:
      - id: "<id>"
        status: loaded_not_imported
  catalogs: [<ids>]

comparisons:
  - requirement_id: "<req-id>"
    label: "<label>"
    policy_value: "<value>"
    policy_source: "<policy-id>"
    requirement_text: "<text>"
    catalog_source: "<catalog-id>"
    interpretation: "<your assessment>"
    resolution: "<user decision>"
```

## MCP Resources and Tools

- `complypack://catalog/*` — Control Catalogs, Guidance Catalogs, Policies
- `complypack://mapping/*` — Mapping Documents
- `analyze_parameter_delta` — gather L3 parameter values alongside L1/L2 requirement text
- `get_assessment_requirements` — get requirement details, supports scope filtering

**DO NOT parse local YAML files to extract control data.** All control IDs, requirement IDs, requirement text, and parameter values MUST come from MCP resources or tools.

## Red Flags

- [ ] Did you classify guidance as mandated vs. under evaluation?
- [ ] Did the user decide every item where values differ?
- [ ] Did you ensure no resolution goes below the parent Policy floor?
- [ ] Did every value come from MCP?
- [ ] Did you use `analyze_parameter_delta` to gather comparisons, not parse files?
