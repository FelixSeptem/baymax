## 1. Negotiation Core and Strategy Semantics

- [x] 1.1 Add capability negotiation core types for required/optional capability sets and strategy modes.
- [x] 1.2 Implement negotiation engine with default `fail_fast` and request-level `best_effort` override handling.
- [x] 1.3 Implement deterministic reason taxonomy emission for required-missing, optional-downgrade, and strategy-override paths.
- [x] 1.4 Add unit tests for negotiation matrix, taxonomy stability, and invalid strategy handling.

## 2. Runtime Integration and Run/Stream Equivalence

- [x] 2.1 Integrate negotiation engine into runtime adapter invocation boundary.
- [x] 2.2 Enforce fail-fast rejection on missing required capability.
- [x] 2.3 Enforce deterministic downgrade on missing optional capability under downgrade-allowed strategy.
- [x] 2.4 Add Run/Stream equivalence tests for negotiation acceptance, rejection, and downgrade outcomes.
- [x] 2.5 Add negotiation diagnostics fields and verify additive+nullable+default compatibility behavior.

## 3. Scaffold and Conformance Alignment

- [x] 3.1 Extend adapter scaffold generator to include negotiation/fallback test skeleton and default strategy config.
- [x] 3.2 Add request-level strategy override hook in generated scaffold samples.
- [x] 3.3 Extend adapter conformance harness with capability negotiation matrix cases.
- [x] 3.4 Add conformance checks for profile alignment between declared adapter capability shape and executed negotiation strategy.

## 4. Gate Integration

- [x] 4.1 Add `scripts/check-adapter-capability-contract.sh` for negotiation contract validation.
- [x] 4.2 Add `scripts/check-adapter-capability-contract.ps1` with parity to shell behavior.
- [x] 4.3 Integrate capability contract checks into `scripts/check-quality-gate.sh`.
- [x] 4.4 Integrate capability contract checks into `scripts/check-quality-gate.ps1`.
- [x] 4.5 Ensure capability contract failures are fail-fast and non-zero.

## 5. Documentation and Traceability

- [x] 5.1 Update `README.md` with capability negotiation behavior and strategy defaults.
- [x] 5.2 Update `docs/external-adapter-template-index.md` with scaffold negotiation guidance.
- [x] 5.3 Update `docs/adapter-migration-mapping.md` with capability negotiation migration notes.
- [x] 5.4 Update `docs/runtime-config-diagnostics.md` with negotiation diagnostics fields and reason taxonomy.
- [x] 5.5 Update `docs/mainline-contract-test-index.md` with capability negotiation contract/gate mappings.
- [x] 5.6 Update `docs/development-roadmap.md` with A27 scope and sequencing note.

## 6. Validation

- [x] 6.1 Run `go test ./...`.
- [x] 6.2 Run `go test -race ./...`.
- [x] 6.3 Run `golangci-lint run --config .golangci.yml`.
- [x] 6.4 Run `bash scripts/check-adapter-capability-contract.sh` and `pwsh -File scripts/check-adapter-capability-contract.ps1`.
- [x] 6.5 Run `bash scripts/check-quality-gate.sh` and `pwsh -File scripts/check-quality-gate.ps1`.
- [x] 6.6 Run `openspec validate introduce-adapter-capability-negotiation-and-fallback-contract-a27 --strict`.
