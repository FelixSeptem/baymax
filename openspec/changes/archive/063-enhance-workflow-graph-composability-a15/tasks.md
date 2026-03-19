## 1. DSL and Compiler Foundation

- [x] 1.1 Add workflow DSL schema for `subgraphs`, `use_subgraph`, `condition_templates`, and `template_vars`.
- [x] 1.2 Implement workflow graph compiler that expands subgraphs before planning.
- [x] 1.3 Enforce subgraph recursion depth limit `3` in compiler validation.
- [x] 1.4 Implement canonical expanded step ID format `<subgraph_alias>/<step_id>`.
- [x] 1.5 Add deterministic collision detection for alias and expanded step IDs.

## 2. Validation and Override Semantics

- [x] 2.1 Enforce condition-template scope to `condition` only.
- [x] 2.2 Enforce subgraph override policy: allow `retry` and `timeout`, reject `kind`.
- [x] 2.3 Add fail-fast validation for missing template, missing variable, cycle reference, depth overflow, and forbidden override.
- [x] 2.4 Keep compile failures at pre-dispatch boundary for Run and Stream.

## 3. Engine and Composed Integration

- [x] 3.1 Route expanded workflow definition into existing deterministic planner and executor without changing terminal semantics.
- [x] 3.2 Ensure checkpoint/resume preserves expanded-step deterministic continuation.
- [x] 3.3 Ensure composer-managed workflow path supports expanded remote A2A steps.
- [x] 3.4 Add Run/Stream equivalence tests for composable workflow in composed orchestration path.

## 4. Config Diagnostics and Compatibility

- [x] 4.1 Add runtime feature flag `workflow.graph_composability.enabled` with default `false` and precedence `env > file > default`.
- [x] 4.2 Add additive diagnostics fields: `workflow_subgraph_expansion_total`, `workflow_condition_template_total`, `workflow_graph_compile_failed`.
- [x] 4.3 Add parser compatibility tests for `additive + nullable + default` semantics on new diagnostics fields.
- [x] 4.4 Ensure disabled-flag behavior preserves legacy flat workflow DSL semantics.

## 5. Contract Tests and Quality Gate

- [x] 5.1 Add contract tests for expansion determinism and canonical expanded IDs.
- [x] 5.2 Add contract tests for compile fail-fast matrix (depth, cycles, template scope, override policy).
- [x] 5.3 Add contract tests for composable workflow Run/Stream equivalence and resume consistency.
- [x] 5.4 Integrate A15 suites into `check-multi-agent-shared-contract.sh` and `.ps1`.
- [x] 5.5 Update `tool/contributioncheck` mappings and assertions for A15 contract index coverage.

## 6. Documentation and Validation

- [x] 6.1 Update `README.md` with minimal composable workflow DSL example and feature-flag note.
- [x] 6.2 Update `docs/runtime-config-diagnostics.md` for new config and diagnostics fields.
- [x] 6.3 Update `docs/mainline-contract-test-index.md` with A15 mapping rows.
- [x] 6.4 Update `docs/development-roadmap.md` with A15 scope and sequencing after A14.
- [x] 6.5 Run `go test ./...`.
- [x] 6.6 Run `go test -race ./...`.
- [x] 6.7 Run `golangci-lint run --config .golangci.yml`.
- [x] 6.8 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 6.9 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.

