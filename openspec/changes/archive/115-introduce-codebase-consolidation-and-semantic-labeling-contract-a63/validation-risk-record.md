# A63 Validation and Risk Record (2026-04-06)

## Executed Validation

- `go test ./... -timeout 15m`
- `go test -race ./... -timeout 20m`
- `golangci-lint run --config .golangci.yml`
- `pwsh -File scripts/check-docs-consistency.ps1`
- `pwsh -File scripts/check-semantic-labeling-governance.ps1`
- `pwsh -File scripts/check-full-chain-example-smoke.ps1`
- `BAYMAX_QUALITY_GATE_SCOPE=general pwsh -File scripts/check-quality-gate.ps1`
- `pwsh -File scripts/check-go-file-line-budget.ps1`
- `BAYMAX_QUALITY_GATE_SCOPE=full BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=3600 BAYMAX_QUALITY_GATE_STEP_TIMEOUT_SECONDS=1800 pwsh -File scripts/check-quality-gate.ps1`
- Focused regressions:
  - `go test ./tool/diagnosticsreplay -count=1 -timeout 10m`
  - `go test ./integration -run '^(TestPrimaryReasonArbitrationReplayContract|TestReplayContract|TestTailGovernance|TestWorkflowGraphComposability)' -count=1 -timeout 10m`
  - `go test ./core/runner ./runtime/config -run 'ReactPlanHook|RealtimeRunStream|RuntimeContextJIT|RuntimeReactPlanNotebook|RuntimeRealtime' -count=1 -timeout 10m`
  - `go test ./tool/contributioncheck -run 'TestMainlineContractIndexReferencesExistingTests' -count=1 -timeout 5m`

## Gate Result and Unexecuted/Blocked Item

- `pwsh -File scripts/check-quality-gate.ps1` was executed in both `general` and `full` scope.
- `full` scope passed with explicit timeout budget (`total=3600s`, `step=1800s`).
- Slowest step in final full run: `context jit organization contract suites` = `196.11s` (well below default `600s` step timeout).
- No remaining blocked validation item for A63 convergence in this session.

## Risk Points

- Naming migration now uses semantic-primary fixture/test identifiers for realtime and context-jit suites while preserving legacy fixture read-compatibility; risk is low and bounded to test/replay naming.
- Full-chain example removed legacy `A20_*` marker emission and smoke fallback; downstream scripts parsing only legacy markers will need to switch to `FULL_CHAIN_*`.

## Rollback Points

- Replay fixture compatibility bridge:
  - `tool/diagnosticsreplay/replay_test.go` `semanticFixtureLegacyAliases`
  - `integration/primary_reason_arbitration_replay_contract_test.go` `semanticFixtureLegacyAliases`
  These maps allow deterministic fallback to legacy fixture files without changing replay semantics.
- Full-chain marker rollback is isolated to:
  - `examples/09-multi-agent-full-chain-reference/main.go`
  - `scripts/check-full-chain-example-smoke.sh`
  - `scripts/check-full-chain-example-smoke.ps1`
  - `examples/09-multi-agent-full-chain-reference/README.md`

## Migration Impact

- Test names, fixture references, and env prefixes for A67/A68-context/realtime paths are semanticized.
- Contract index references are updated to semantic test function names.
- Go file line-budget exception baselines were synchronized for `core/runner/runner.go` and `runtime/config/config.go` to match current staged split backlog.
- No runtime API semantics were changed; config precedence and replay taxonomy behavior remain unchanged.

## Convergence Update (Task 2.1 Completion)

- Context Assembler active implementation symbols were further semanticized in `context/assembler`:
  - `applyCA2` -> `applyStage2RoutingAndDisclosure`
  - `applyCA3` -> `applyPressureCompactionAndSwapback`
  - internal CA3-prefixed run-state/decision/zone/compactor symbols renamed to pressure-compaction semantic naming.
- Additional validations executed after this refactor:
  - `go test ./context/assembler ./core/runner ./runtime/config ./runtime/diagnostics ./observability/event -count=1`
  - `golangci-lint run --config .golangci.yml`
  - `pwsh -File scripts/check-semantic-labeling-governance.ps1`
  - `pwsh -File scripts/check-docs-consistency.ps1`
  - `BAYMAX_QUALITY_GATE_SCOPE=general pwsh -File scripts/check-quality-gate.ps1`
  - `BAYMAX_QUALITY_GATE_SCOPE=full BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=3600 BAYMAX_QUALITY_GATE_STEP_TIMEOUT_SECONDS=1800 pwsh -File scripts/check-quality-gate.ps1`
  - `pwsh -File scripts/check-go-file-line-budget.ps1`
- Naming governance summary improved during this pass:
  - `legacy-context-stage-wording-content` reduced from `921` to `859` while keeping gates green.
