## 1. DSL and Compiler Foundation

- [ ] 1.1 Add workflow DSL schema for `subgraphs`, `use_subgraph`, `condition_templates`, and `template_vars`.
- [ ] 1.2 Implement workflow graph compiler that expands subgraphs before planning.
- [ ] 1.3 Enforce subgraph recursion depth limit `3` in compiler validation.
- [ ] 1.4 Implement canonical expanded step ID format `<subgraph_alias>/<step_id>`.
- [ ] 1.5 Add deterministic collision detection for alias and expanded step IDs.

## 2. Validation and Override Semantics

- [ ] 2.1 Enforce condition-template scope to `condition` only.
- [ ] 2.2 Enforce subgraph override policy: allow `retry` and `timeout`, reject `kind`.
- [ ] 2.3 Add fail-fast validation for missing template, missing variable, cycle reference, depth overflow, and forbidden override.
- [ ] 2.4 Keep compile failures at pre-dispatch boundary for Run and Stream.

## 3. Engine and Composed Integration

- [ ] 3.1 Route expanded workflow definition into existing deterministic planner and executor without changing terminal semantics.
- [ ] 3.2 Ensure checkpoint/resume preserves expanded-step deterministic continuation.
- [ ] 3.3 Ensure composer-managed workflow path supports expanded remote A2A steps.
- [ ] 3.4 Add Run/Stream equivalence tests for composable workflow in composed orchestration path.

## 4. Config Diagnostics and Compatibility

- [ ] 4.1 Add runtime feature flag `workflow.graph_composability.enabled` with default `false` and precedence `env > file > default`.
- [ ] 4.2 Add additive diagnostics fields: `workflow_subgraph_expansion_total`, `workflow_condition_template_total`, `workflow_graph_compile_failed`.
- [ ] 4.3 Add parser compatibility tests for `additive + nullable + default` semantics on new diagnostics fields.
- [ ] 4.4 Ensure disabled-flag behavior preserves legacy flat workflow DSL semantics.

## 5. Contract Tests and Quality Gate

- [ ] 5.1 Add contract tests for expansion determinism and canonical expanded IDs.
- [ ] 5.2 Add contract tests for compile fail-fast matrix (depth, cycles, template scope, override policy).
- [ ] 5.3 Add contract tests for composable workflow Run/Stream equivalence and resume consistency.
- [ ] 5.4 Integrate A15 suites into `check-multi-agent-shared-contract.sh` and `.ps1`.
- [ ] 5.5 Update `tool/contributioncheck` mappings and assertions for A15 contract index coverage.

## 6. Documentation and Validation

- [ ] 6.1 Update `README.md` with minimal composable workflow DSL example and feature-flag note.
- [ ] 6.2 Update `docs/runtime-config-diagnostics.md` for new config and diagnostics fields.
- [ ] 6.3 Update `docs/mainline-contract-test-index.md` with A15 mapping rows.
- [ ] 6.4 Update `docs/development-roadmap.md` with A15 scope and sequencing after A14.
- [ ] 6.5 Run `go test ./...`.
- [ ] 6.6 Run `go test -race ./...`.
- [ ] 6.7 Run `golangci-lint run --config .golangci.yml`.
- [ ] 6.8 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 6.9 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
