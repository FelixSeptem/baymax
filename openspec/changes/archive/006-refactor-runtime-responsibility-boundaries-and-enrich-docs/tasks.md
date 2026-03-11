## 1. Runtime Module Refactor

- [x] 1.1 Introduce new global runtime config package (path finalized per design) and move `Config/Manager/Snapshot` ownership there
- [x] 1.2 Introduce new global runtime diagnostics package and move shared diagnostics API ownership there
- [x] 1.3 Split MCP semantics into function-scoped packages (`mcp/profile`, `mcp/retry`, `mcp/diag`) and remove `mcp/runtime`
- [x] 1.4 Remove old path compatibility shims to avoid dual-runtime concepts
- [x] 1.5 Add/adjust import-boundary checks to enforce one-way dependency direction

## 2. Cross-Module Integration

- [x] 2.1 Migrate `mcp/http` and `mcp/stdio` to consume the new global runtime config + diagnostics interfaces
- [x] 2.2 Integrate runner/tool/skill/observability entry points with the global runtime config snapshot API
- [x] 2.3 Integrate runner/tool/skill/observability entry points with the global runtime diagnostics API
- [x] 2.4 Verify semantic parity for precedence, fail-fast validation, and hot-reload behavior after package relocation
- [x] 2.5 Ensure diagnostics APIs still return bounded, normalized records across modules (including skill lifecycle records)

## 3. Documentation Expansion

- [x] 3.1 Add architecture document covering module boundaries, dependency direction, and ownership map
- [x] 3.2 Add runtime config field index and env mapping reference with old-to-new package migration table
- [x] 3.3 Update README and docs navigation with migration guide, FAQ, and extension constraints
- [x] 3.4 Add example snippets showing new package usage and legacy-to-new migration patterns

## 4. Validation and Quality Gates

- [x] 4.1 Add tests for migration compatibility (old/new path semantic equivalence)
- [x] 4.2 Add tests for diagnostics API migration compatibility (field set, ordering semantics, bounded history, sanitization)
- [x] 4.3 Add tests for skill diagnostics compatibility (discovery/trigger/compile/failure semantics)
- [x] 4.4 Add dependency-boundary checks in CI to prevent reverse coupling regressions
- [x] 4.5 Run `go test ./...` and supported `-race` checks; document any environment-specific caveats
- [x] 4.6 Run `golangci-lint` with updated config and ensure docs consistency checks pass
