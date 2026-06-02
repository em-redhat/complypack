# ADR 006: CUE Schema Contract Validation at Authoring Time

**Status:** Accepted

**Date:** 2026-06-01

**Context:**

Rego policies reference `input.*` paths (e.g., `input.metadata.name`, `input.spec.containers[*].image`) based on the structure of resources they evaluate. If a policy references `input.nonexistent.field` for a platform that doesn't have that field, the policy:

1. **Silently fails at runtime**: OPA evaluates the reference as `undefined`, causing logic errors
2. **Provides no feedback during authoring**: Author only discovers the issue when running policies against real resources
3. **Creates maintenance burden**: Schema changes (field renames, deprecations) break policies without warning

Traditional approaches:

- **Runtime-only validation**: Wait for OPA to evaluate, check if violations occur (too late)
- **Manual schema inspection**: Author cross-references OpenAPI/JSON Schema docs (error-prone)
- **Static analysis on outputs**: Analyze OPA's AST for undefined refs (misses schema evolution)

We need **authoring-time contract validation** that catches schema mismatches before policies are packaged or deployed.

**Decision:**

We implement **CUE schema contract validation** in the `CheckContract` evaluator method:

```go
func CheckContract(filename, src string, schema cue.Value) ([]ContractViolation, error)
```

**How It Works:**

1. **Parse Rego AST**: Use `ast.ParseModuleWithOpts` with `RegoV1`
2. **Extract `input.*` references**: Walk AST with `ast.WalkRefs` to find all `input.*` paths
3. **Validate against CUE schema**: For each path, traverse the CUE value using `Fields(cue.All())` iterator
4. **Report violations**: Return `ContractViolation{Path, Location}` for undefined paths

**Example:**

```rego
package kubernetes
import rego.v1

deny_privileged contains msg if {
    input.kind == "Pod"
    input.spec.containers[*].securityContext.privileged == true
    msg := "Privileged container detected"
}
```

Validated against `schemas/cue/kubernetes.cue`:

```cue
apiVersion?: string
kind?: string
metadata?: #Metadata
spec?: #Spec

#Spec: {
    containers?: [...#Container]
}

#Container: {
    securityContext?: #SecurityContext
}

#SecurityContext: {
    privileged?: bool
}
```

✅ **All references valid** → `CheckContract` returns `[]`

If policy references `input.spec.invalidField`:

❌ **Violation detected** → `ContractViolation{Path: "input.spec.invalidField", Location: "policy.rego:5"}`

**Why CUE?**

1. **Precision**: CUE schemas define exact field names, types, and optionality
2. **Composability**: CUE definitions (`#Container`, `#Metadata`) match Kubernetes/Terraform structure naturally
3. **Existing investment**: ComplyPack already uses CUE schemas for built-in platforms
4. **Path validation**: CUE's `Fields()` iterator handles optional fields (`field?: type`) correctly

**Limitations:**

- **Dynamic references skipped**: `input[variable]` or `input.metadata[key]` can't be validated statically
- **Schema maintenance**: Schemas must stay in sync with platform APIs (same challenge as OpenAPI specs)
- **No type checking**: Only validates path existence, not value types (e.g., `string` vs `int`)

**Implementation:**

- **`internal/validator/contract.go`**:
  - `extractInputRefs()`: Walks OPA AST to find all `input.*` refs
  - `pathExistsInSchema()`: Iterates CUE fields to validate paths (handles optional fields)
  - `buildPath()`: Converts AST references to dotted notation
- **`schemas/embed.go`**:
  - `GetBuiltInCUESchema()`: Loads CUE source files from `schemas/cue/*.cue`
  - Existing: `GetBuiltInSchema()` for JSON Schema (backward compat)

**Consequences:**

**Benefits:**

- **Early feedback**: Authors catch schema mismatches during policy development, not in production
- **Safe refactoring**: Renaming a schema field surfaces all affected policies immediately
- **Documentation as code**: CUE schemas serve as machine-readable contracts
- **CI/CD integration**: Contract validation can run in pre-commit hooks or CI pipelines

**Drawbacks:**

- **Schema drift**: If CUE schemas aren't updated with platform changes, validation gives false confidence
- **Dynamic reference blind spots**: Policies using computed paths bypass validation
- **Learning curve**: Authors need to understand CUE schema format (mitigated by built-in schemas)

**Future Enhancements:**

- **Type checking**: Extend to validate value types (requires CUE unification with policy logic)
- **Auto-generated schemas**: Generate CUE from Kubernetes OpenAPI specs (similar to `cue import openapi`)
- **Partial validation**: Flag dynamic refs with warnings instead of skipping silently

**Related:**

- ADR 005: Evaluator Interface Pattern
- Issue #10: CLI validation workflow
- `schemas/cue/kubernetes.cue`: Reference implementation
