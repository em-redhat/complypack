---
name: comply-adherence
description: Populate a Gemara Policy defining what controls apply, with what parameter values, and how evidence will be collected
user-invocable: false
---

# Adherence — Compile Policy

Compile or alter a Gemara Policy artifact. Declares what controls apply, with exact parameter values, and defines the adherence plan: frequency, evaluation method, and evidence requirements.

Evidence collection and Evaluation Logs are produced by `complyctl` at runtime.

## Prerequisites

- `.complytime/scoping.yaml`
- `.complytime/delta-report.yaml`

Verify all parameters are resolved (no `pending_user_decision`). If unresolved, tell the user to re-run mapping.

## Process

### Step 1: Read Input Artifacts

Read both `.complytime/scoping.yaml` and `.complytime/delta-report.yaml`.

### Step 2: Build Mapping References

From the delta report's `sources`, build `mapping_references`:
- One for the parent Policy
- One for each scoped Control Catalog
- One for each Guidance Catalog for the target framework

### Step 3: Build Imports

- `imports.catalogs` — one per Control Catalog
- `imports.guidance` — one per mandated Guidance Catalog

### Step 4: Build Assessment Plans

Group by `requirement-id`. Each plan:
- `id` — unique plan identifier
- `requirement-id` — the assessment requirement this plan addresses
- `frequency` (e.g., "30d", "90d", "365d")
- `evaluation-methods` — list of `{id, type: Behavioral|Intent, mode: Automated|Manual}`
- `evidence-requirements` — what evidence is collected
- `parameters` — frozen values from harmonization, each with `id`, `label`, `accepted-values`, `description`

### Step 5: Compile the Policy

```yaml
title: "<System Name> Policy"
metadata:
  id: <system-name>-policy
  gemara-version: v1.0.0
  type: Policy
  description: "<system description>"
  author:
    id: <author-id>
    name: <author-name>
    type: Software Assisted
  mapping-references:
    - id: <ref-id>
      title: "<reference title>"
      version: "<version>"
contacts:
  responsible:
    - name: "<contact name>"
  accountable:
    - name: "<contact name>"
scope:
  in:
    technologies:
      - <technology from system profile>
    groups:
      - <applicability group, e.g. maturity-1, maturity-2>
imports:
  catalogs:
    - reference-id: <ref-id>
  guidance:
    - reference-id: <ref-id>
adherence:
  assessment-plans:
    - id: <plan-id>
      requirement-id: "<req-id>"
      frequency: "<cadence>"
      evaluation-methods:
        - id: <method-id>
          type: Behavioral
          mode: Automated
      evidence-requirements: "<what>"
      parameters:
        - id: <param-id>
          label: "<label>"
          accepted-values: ["<value>"]
          description: "<rationale>"
```

Use `scope.in.groups` for applicability groups from the scoping stage (e.g., maturity levels, risk tiers). This scopes the policy to only the controls that match those groups.

### Step 6: Validate

Write `.complytime/child-policy.yaml`. If the Gemara MCP server is available, validate against the schema with the Policy definition.

### Step 7: Present Summary

Show mapping references, imported catalogs/guidance, assessment plans count, and audit strengths.

> "Policy written to `.complytime/child-policy.yaml`. Invoke `/comply:pack` to generate assessment logic."

## MCP Resources and Tools

- `complypack://catalog/*` — Control Catalogs, Guidance Catalogs, Policies
- `get_assessment_requirements` — get requirement details and parameters for building assessment plans

**DO NOT parse local YAML files to extract control data.** Use `get_assessment_requirements` to get requirement text, parameters, and applicability. All control data MUST come from MCP resources or tools.

## Red Flags

- [ ] Any unresolved parameters?
- [ ] Every mapping reference has a corresponding import?
- [ ] Every assessment plan has frequency and evaluation method?
- [ ] Every value came from MCP or the delta report?
- [ ] Did you use `get_assessment_requirements`, not parse files?
