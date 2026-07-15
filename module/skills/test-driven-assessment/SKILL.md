---
name: test-driven-assessment
user-invocable: false
description: Use before generating any Rego policy or assessment logic — generates human-reviewable test cases from assessment requirements before any policy is written
---

# Test-Driven Assessment

> **Internal skill** — invoked by `comply:build-assessment`, not directly by users. Use `/comply:build-assessment` or `/comply:build-assessment single` instead.

Write test cases first. Get human approval. Then generate policy to satisfy them.

**Core principle:** If a practitioner didn't review the test scenarios, the policy is unverified — regardless of whether it passes `validate_policy`.

## When to Use

**Always before `comply:pack-assessment`.** This skill produces the test cases that pack-assessment's policy must satisfy.

**Generated policy before the tests? Delete it.** The policy is untrusted without approved test cases.

## Process

### Step 1: Read Requirements from MCP

For each automated requirement (from `get_automation_triage`):

1. Read the control definition from the catalog (`complypack://catalog/*`)
2. Call `get_assessment_requirements` to extract parameter values and thresholds
3. Read the platform schema (`complypack://schema/*`) for valid `input.*` paths

**DO NOT proceed without MCP.** If MCP is unreachable, stop. Test cases generated from general knowledge are unverifiable and defeat the purpose.

### Step 2: Generate Test Cases

For each requirement, generate:
- One `test_deny_*` function — input that **should trigger a violation**
- One `test_allow_*` function — input that **should pass cleanly**

### Rules

- Use `with input as {...}` to provide inline test data
- Test data JSON must use `input.*` paths from the platform schema
- Parameter values come from `get_assessment_requirements`, not from sample data
- Each test function gets a one-line comment stating what it checks in plain language
- Name test files `<requirement>_test.rego` in the same package as the policy

### Example

For a requirement "containers must not run as root" with platform `kubernetes-pod`:

```rego
package policy

import rego.v1

# Container runs as root — should deny
test_deny_root_container if {
    result := deny with input as {
        "kind": "Pod",
        "metadata": {"name": "test-pod"},
        "spec": {"containers": [{"name": "app", "securityContext": {"runAsNonRoot": false}}]}
    }
    count(result) > 0
}

# Container runs as non-root — should allow
test_allow_nonroot_container if {
    result := deny with input as {
        "kind": "Pod",
        "metadata": {"name": "test-pod"},
        "spec": {"containers": [{"name": "app", "securityContext": {"runAsNonRoot": true}}]}
    }
    count(result) == 0
}
```

### Step 3: Present for Review

Present each test case to the user with:
- The requirement ID and title
- Plain-language description of the fail scenario
- Plain-language description of the pass scenario
- The test data used in each

**DO NOT proceed to policy generation until the user confirms the test scenarios are correct.**

If the user identifies wrong conditions, missing scenarios, or incorrect parameter values — revise and re-present.

### Step 4: Write to Disk

Save approved test files to `policy/<requirement>_test.rego`.

### Step 5: Hand Off to pack-assessment

The approved test cases are the specification. Invoke `comply:pack-assessment` to generate policy that satisfies them.

When pack-assessment runs `test_policy`, it combines the policy and test content in the same package. **If tests fail, the policy is revised — not the tests.** The tests were human-approved and represent the requirement.

## Red Flags — STOP AND FIX IF THERE ARE ISSUES

- [ ] MCP is unreachable → **STOP.** Do not generate test cases from general knowledge. Inform the user and wait.
- [ ] Policy was generated before test cases → **STOP.** Delete the policy. Generate test cases first.
- [ ] User has not approved the test scenarios → **STOP.** Do not proceed to `comply:pack-assessment`. Present tests and wait for approval.
- [ ] Test cases use parameter values not from `get_assessment_requirements` → **STOP.** Replace with MCP-sourced values.
- [ ] `input.*` paths in test data do not exist in the platform schema → **STOP.** Fix paths to match `complypack://schema/*`.
- [ ] Approved test cases were modified to make policy pass → **STOP.** Revert the tests. Fix the policy instead.
- [ ] Missing a deny or allow test for a requirement → **STOP.** Add the missing test case before proceeding.
