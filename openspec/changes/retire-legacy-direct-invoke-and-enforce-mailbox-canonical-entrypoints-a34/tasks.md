## 1. API Surface Cutover

- [ ] 1.1 Remove legacy direct public invoke entrypoints (`invoke.InvokeSync`, `invoke.InvokeAsync`) from supported API surface.
- [ ] 1.2 Refactor `MailboxBridge` to use package-private helpers so bridge no longer depends on deprecated exported invoke symbols.
- [ ] 1.3 Ensure sync/async invoke error normalization and retryable hints remain behaviorally equivalent after cutover.

## 2. Call-Site Migration

- [ ] 2.1 Update orchestration call sites (`collab`, `scheduler`, and related adapters) to consume mailbox canonical invoke entrypoints only.
- [ ] 2.2 Remove or rewrite tests that assert legacy direct invoke as canonical behavior.
- [ ] 2.3 Add/adjust unit tests for mailbox canonical invoke helper behavior and validation errors.

## 3. Contract Tests and Shared Gate

- [ ] 3.1 Update sync invocation contract suites to assert canonical mailbox-only public entrypoints.
- [ ] 3.2 Update async reporting contract suites to assert legacy direct async invoke path is no longer supported public contract.
- [ ] 3.3 Add/adjust mailbox contract suites to enforce sync/async/delayed entrypoint convergence through mailbox path.
- [ ] 3.4 Keep Run/Stream equivalence, memory/file parity, and replay idempotency suites green after cutover.

## 4. Canonical-Only Quality Gate

- [ ] 4.1 Add canonical-only regression checks into `scripts/check-multi-agent-shared-contract.sh` and `.ps1`.
- [ ] 4.2 Add canonical-only regression checks into `scripts/check-quality-gate.sh` and `.ps1`.
- [ ] 4.3 Ensure gate checks fail fast on legacy direct invoke reintroduction and provide deterministic non-zero exit.

## 5. Documentation Alignment

- [ ] 5.1 Update `README.md` to state mailbox bridge is the only canonical invoke entrypoint for sync/async/delayed.
- [ ] 5.2 Update `docs/mainline-contract-test-index.md` to remove transition wording and map canonical-only coverage/gate paths.
- [ ] 5.3 Update `docs/development-roadmap.md` and `orchestration/README.md` to remove deprecated-but-active middle-state wording.

## 6. Repo-Wide Deprecated Usage Audit

- [ ] 6.1 Run repository-wide scan for deprecated symbols and classify each as active-use, test-only, or doc-only.
- [ ] 6.2 For active-use deprecated symbols, either migrate usage in this change or explicitly record non-goal with follow-up.
- [ ] 6.3 Add/update a lightweight check to prevent reintroducing deprecated-in-use invoke patterns.

## 7. Validation

- [ ] 7.1 Run `go test ./...`.
- [ ] 7.2 Run `go test -race ./...`.
- [ ] 7.3 Run `golangci-lint run --config .golangci.yml`.
- [ ] 7.4 Run `pwsh -File scripts/check-multi-agent-shared-contract.ps1`.
- [ ] 7.5 Run `pwsh -File scripts/check-quality-gate.ps1`.
- [ ] 7.6 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 7.7 Run `openspec validate retire-legacy-direct-invoke-and-enforce-mailbox-canonical-entrypoints-a34 --strict`.
