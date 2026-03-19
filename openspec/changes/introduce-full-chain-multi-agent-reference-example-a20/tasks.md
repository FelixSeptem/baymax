## 1. Full-Chain Example Scaffold

- [ ] 1.1 Create `examples/09-multi-agent-full-chain-reference` entrypoint and README scaffold.
- [ ] 1.2 Compose `teams + workflow + a2a + scheduler + recovery` in one runnable baseline path.
- [ ] 1.3 Ensure default path uses in-memory A2A and requires no external services.

## 2. Run and Stream Coverage

- [ ] 2.1 Implement full-chain Run path with deterministic terminal summary output.
- [ ] 2.2 Implement full-chain Stream path with streaming output and terminal convergence markers.
- [ ] 2.3 Ensure Run/Stream outputs remain semantically aligned for the same scenario intent.

## 3. Async, Delayed, and Recovery Composition

- [ ] 3.1 Add at least one async-reporting checkpoint in the example flow.
- [ ] 3.2 Add at least one delayed-dispatch checkpoint in the example flow.
- [ ] 3.3 Add recovery-enabled path and expose minimal recovery markers in example output.
- [ ] 3.4 Document checkpoint meanings and verification guidance in example README.

## 4. Smoke Gate Integration

- [ ] 4.1 Add full-chain example smoke command/script with required success marker assertions.
- [ ] 4.2 Integrate smoke validation into `scripts/check-quality-gate.sh`.
- [ ] 4.3 Integrate smoke validation into `scripts/check-quality-gate.ps1`.
- [ ] 4.4 Ensure smoke failures are fail-fast and block validation.

## 5. Documentation and Index Alignment

- [ ] 5.1 Update root `README.md` tutorial navigation with full-chain example entry.
- [ ] 5.2 Update `docs/mainline-contract-test-index.md` with full-chain example smoke traceability row.
- [ ] 5.3 Update `docs/development-roadmap.md` with A20 scope and sequencing notes.

## 6. Validation

- [ ] 6.1 Run `go run ./examples/09-multi-agent-full-chain-reference`.
- [ ] 6.2 Run `go test ./...`.
- [ ] 6.3 Run `go test -race ./...`.
- [ ] 6.4 Run `golangci-lint run --config .golangci.yml`.
- [ ] 6.5 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 6.6 Run `bash scripts/check-quality-gate.sh` and `pwsh -File scripts/check-quality-gate.ps1`.
- [ ] 6.7 Run `openspec validate introduce-full-chain-multi-agent-reference-example-a20 --strict`.

