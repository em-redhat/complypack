---
name: scoping
description: Characterize a system's component stack, scope existing Control Catalogs, and identify coverage gaps
user-invocable: false
---

# Scoping

Characterize the system, identify its technology component stack, and scope which existing Control Catalogs apply. Produces `.complytime/scoping.yaml`.

## Modes

- **Conversational:** Ask structured questions one at a time.
- **Document ingestion:** Extract and normalize from existing SAR, architecture doc, deployment manifest, or IaC.
- **Hybrid:** Fill gaps with targeted questions.

## Process

### Step 1: Characterize the System

1. **Identity:** Name, description, purpose, criticality
2. **Data classification:** public, internal, confidential, restricted
3. **Component stack:** Name, type, runtime/engine, image/version
4. **Data flows:** Source → destination, protocol, data types
5. **Deployment:** Cloud/on-prem/hybrid, provider, regions
6. **Boundaries:** DMZ, internal, external-facing components

### Step 2: Scope Control Catalogs

Query MCP for available Control Catalogs:

```text
ListMcpResourcesTool(server="complypack")
```

Map each component to an existing Control Catalog. No matching catalog → record as a **gap**.

### Step 3: Filter by Applicability Group

If the user specified a scoping level (e.g., maturity level, risk tier), use `get_assessment_requirements` with the `scope` parameter to get only the requirements that apply.

**Important:** The `catalogName` parameter takes a **policy name**, not a catalog name. The policy resolves its imported catalogs internally. Find the policy name from `ListMcpResourcesTool` output — look for resources with `kind: Policy`.

```text
CallMcpToolTool(server="complypack", tool="get_assessment_requirements", arguments={"catalogName": "<policy-name>", "scope": ["<applicability-group>"]})
```

For example, for OSPS Baseline at Maturity Level 2 with a policy named `example-foundation-policy`:

```text
CallMcpToolTool(server="complypack", tool="get_assessment_requirements", arguments={"catalogName": "example-foundation-policy", "scope": ["maturity-1", "maturity-2"]})
```

This returns only the requirements whose applicability includes the target level. Use this filtered set for the rest of the pipeline — do NOT manually read and parse the catalog to determine which controls apply.

### Step 4: Check for Mapping Documents

Ask the user if they have Gemara Mapping Documents. Check MCP:

```text
ListMcpResourcesTool(server="complypack")
```

Record any available Mapping Documents — the mapping stage will use them.

### Step 5: Identify Gaps

For each component with no matching Control Catalog:
- Record the component, its type, and engine
- Recommend: "Control Catalog needed for this technology, or accept risk"

### Step 6: Confirm with User

Present findings. Ask user to confirm or adjust before writing.

### Step 7: Write Output

Write `.complytime/scoping.yaml`:

```yaml
version: "1"
created: YYYY-MM-DD
system:
  name: "<name>"
  description: "<description>"
  criticality: <high|medium|low>
  data_classification: <public|internal|confidential|restricted>
  components:
    - name: "<name>"
      type: <container|database|cache|api-gateway|model-server|...>
      runtime: "<runtime>"
  data_flows:
    - from: "<source>"
      to: "<destination>"
      protocol: <tls|mtls|plaintext>
      data_types: [<types>]
  deployment:
    model: <cloud|on-prem|hybrid>
    provider: "<provider>"
    regions: [<regions>]

catalog_scope:
  covered:
    - catalog_id: "<id>"
      components: [<names>]
      reason: "<why>"
  gaps:
    - component: "<name>"
      type: "<type>"
      finding: "No Control Catalog available"

mapping_documents:
  - id: "<id>"
    source: "<where found>"
```

## MCP Resources and Tools

- `complypack://catalog/*` — Control Catalogs, Guidance Catalogs, Policies
- `complypack://mapping/*` — Mapping Documents
- `get_assessment_requirements` — get requirement details and parameters for a specific control or in batch

**DO NOT parse local YAML files to extract control data.** All control IDs, requirement IDs, requirement text, and parameter values MUST come from MCP resources or tools.

## Red Flags

- [ ] Did you query MCP for catalogs, not assume them?
- [ ] Did you use `get_assessment_requirements` for requirement details, not parse files?
- [ ] Did you record gaps?
- [ ] Did you confirm with the user before writing?
