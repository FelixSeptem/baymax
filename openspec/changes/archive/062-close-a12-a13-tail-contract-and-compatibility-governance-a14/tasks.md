## 1. Freeze Inputs and Scope Alignment

- [x] 1.1 Freeze A12/A13 closure baseline (reason taxonomy, additive fields, required correlations) and record snapshot source.
- [x] 1.2 Define required cross-mode matrix rows for `sync/async/delayed × Run/Stream × qos/recovery` key paths.
- [x] 1.3 Confirm A14 scope excludes new runtime features and focuses on contract/gate/doc convergence only.

## 2. Shared Contract and Gate Convergence

- [x] 2.1 Extend `tool/contributioncheck/multi_agent_contract.go` required reason set to include delayed canonical reasons.
- [x] 2.2 Add/adjust contribution-check tests to assert async+delayed completeness and missing-reason failure codes.
- [x] 2.3 Update `scripts/check-multi-agent-shared-contract.sh` and `.ps1` to include A14 closure matrix suites.
- [x] 2.4 Ensure shared gate keeps single blocking path (no parallel disconnected gate for the same contract domain).

## 3. Contract Tests and Compatibility Semantics

- [x] 3.1 Add integration contract cases for cross-mode matrix core rows (sync/async/delayed in Run and Stream).
- [x] 3.2 Add qos/recovery key-path matrix assertions for semantic equivalence and replay-idempotency.
- [x] 3.3 Add diagnostics parser compatibility tests for `additive + nullable + default` on A12/A13 additive fields.
- [x] 3.4 Ensure combined async+delayed duplicate replay does not inflate logical run aggregates.

## 4. Documentation and Contract Index Alignment

- [x] 4.1 Update `docs/runtime-config-diagnostics.md` with unified A12/A13 compatibility-window and parser semantics.
- [x] 4.2 Update `docs/mainline-contract-test-index.md` with A14 cross-mode matrix coverage mapping.
- [x] 4.3 Update `docs/development-roadmap.md` to reflect A12/A13 closure sequence and A14 status.
- [x] 4.4 Verify docs wording, gate scope, and index rows are mutually consistent in one change set.

## 5. Validation and Closure

- [x] 5.1 Run `go test ./...`.
- [x] 5.2 Run `go test -race ./...`.
- [x] 5.3 Run `golangci-lint run --config .golangci.yml`.
- [x] 5.4 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 5.5 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [x] 5.6 Run `openspec validate close-a12-a13-tail-contract-and-compatibility-governance-a14 --strict` and resolve all findings.
