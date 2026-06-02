# ADR 005: Evaluator Interface Pattern for Policy-Language Abstraction

**Status:** Accepted

**Date:** 2026-06-01

**Context:**

ComplyPack needs to validate, test, and package policies across multiple policy languages (OPA Rego, CEL, Kyverno, etc.). Hard-coding OPA-specific logic throughout the codebase would:

1. Create tight coupling between business logic and policy language implementation
2. Make it difficult to add support for new policy languages
3. Duplicate validation/testing logic across different parts of the system
4. Violate the Open/Closed Principle (open for extension, closed for modification)

We need a policy-language-agnostic design that allows ComplyPack to treat policy content as opaque bytes while delegating language-specific operations to pluggable implementations.

**Decision:**

We adopt the **Evaluator Interface Pattern** with a thread-safe registry:

```go
type Evaluator interface {
    ID() string                           // Unique identifier (e.g., "io.complytime.opa")
    Validate(filename, src string) []error
    CheckContract(filename, src string, schema cue.Value) ([]ContractViolation, error)
    Test(ctx context.Context, files map[string]string) (*TestResults, error)
    Lint(filename, src string) ([]LintWarning, error)
    FileExtension() string                // Expected file extension (e.g., ".rego")
}
```

**Registry Pattern:**

```go
type Registry struct {
    mu         sync.RWMutex
    evaluators map[string]Evaluator
}

func NewRegistry() *Registry
func (r *Registry) Register(e Evaluator)
func (r *Registry) Get(id string) (Evaluator, error)
func (r *Registry) IDs() []string
```

**Key Design Decisions:**

1. **ID-based dispatch**: Policies are tagged with `evaluator-id` in config/metadata, allowing runtime selection
2. **Thread-safe registry**: Supports concurrent access for multi-threaded validation
3. **Graceful degradation**: `Lint()` returns `nil, nil` if linter unavailable (optional tooling)
4. **Context-aware testing**: `Test()` accepts `context.Context` for cancellation/timeout
5. **Schema-agnostic contracts**: `CheckContract` uses `cue.Value` to validate input references against any schema format

**Implementation:**

- **`internal/evaluator/evaluator.go`**: Interface definitions and shared types
- **`internal/evaluator/registry.go`**: Thread-safe registry implementation
- **`internal/evaluator/opa.go`**: OPA implementation (`ID() = "io.complytime.opa"`)
- **`DefaultRegistry()`**: Pre-registers OPA evaluator for convenience

**Consequences:**

**Benefits:**

- **Extensibility**: Adding CEL/Kyverno support requires implementing one interface, no changes to consuming code
- **Testability**: Easy to mock evaluators for unit tests
- **Separation of concerns**: Business logic (validation workflows) decoupled from policy language details
- **Type safety**: Compile-time guarantees that all language implementations support required operations

**Drawbacks:**

- **Interface overhead**: Each language must implement all methods (mitigated by graceful degradation for optional features like `Lint`)
- **Indirection**: One extra layer between caller and implementation (negligible performance cost)

**Future Considerations:**

- **Plugin system**: Could load evaluators from shared libraries (`.so`/`.dylib`) for out-of-tree languages
- **Capability negotiation**: Some languages might not support all operations (e.g., no native testing framework)
- **Performance tuning**: Could add `PrepareForEval()` method for query compilation caching

**Related:**

- ADR 006: CUE Schema Contract Validation
- ADR 007: OPA SDK In-Process vs Subprocess
- Issue #10: CLI for policy validation and packaging
- Issue #12: MCP validation tools
