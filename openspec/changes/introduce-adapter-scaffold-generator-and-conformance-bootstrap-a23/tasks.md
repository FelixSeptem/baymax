## 1. Scaffold Generator Command and Library

- [ ] 1.1 Add scaffold generation package and `cmd/adapter-scaffold` command entry.
- [ ] 1.2 Implement argument contract for `-type`, `-name`, `-output`, and `-force` with fail-fast validation.
- [ ] 1.3 Implement deterministic generation plan builder with preflight conflict detection before write.

## 2. Template Set and File Generation

- [ ] 2.1 Add scaffold templates for `mcp`, `model`, and `tool` categories.
- [ ] 2.2 Generate minimum onboarding artifacts: adapter skeleton, README, unit-test skeleton, and conformance bootstrap skeleton.
- [ ] 2.3 Implement stable placeholder substitution and deterministic file ordering.

## 3. Conformance Bootstrap Alignment

- [ ] 3.1 Wire generated conformance bootstrap skeleton to repository adapter conformance harness entry.
- [ ] 3.2 Add category-specific bootstrap mapping hints so generated scaffolds follow A22 minimum matrix expectations.
- [ ] 3.3 Add regression tests to verify generated bootstrap path remains executable in offline mode.

## 4. Scaffold Drift Gate Integration

- [ ] 4.1 Add `scripts/check-adapter-scaffold-drift.sh` for deterministic scaffold drift validation.
- [ ] 4.2 Add `scripts/check-adapter-scaffold-drift.ps1` with behavior parity to shell flow.
- [ ] 4.3 Integrate scaffold drift checks into `scripts/check-quality-gate.sh` as blocking step.
- [ ] 4.4 Integrate scaffold drift checks into `scripts/check-quality-gate.ps1` as blocking step.
- [ ] 4.5 Ensure drift mismatch exits fail-fast with deterministic non-zero status and explicit classification.

## 5. Documentation and Traceability

- [ ] 5.1 Update `README.md` with scaffold command usage and default output conventions.
- [ ] 5.2 Update `docs/mainline-contract-test-index.md` to map scaffold drift and conformance bootstrap checks.
- [ ] 5.3 Update `docs/development-roadmap.md` with A23 scope and sequencing.

## 6. Validation

- [ ] 6.1 Run `go test ./...`.
- [ ] 6.2 Run `go test -race ./...`.
- [ ] 6.3 Run `golangci-lint run --config .golangci.yml`.
- [ ] 6.4 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 6.5 Run `bash scripts/check-adapter-conformance.sh` and `pwsh -File scripts/check-adapter-conformance.ps1`.
- [ ] 6.6 Run `bash scripts/check-adapter-scaffold-drift.sh` and `pwsh -File scripts/check-adapter-scaffold-drift.ps1`.
- [ ] 6.7 Run `bash scripts/check-quality-gate.sh` and `pwsh -File scripts/check-quality-gate.ps1`.
- [ ] 6.8 Run `openspec validate introduce-adapter-scaffold-generator-and-conformance-bootstrap-a23 --strict`.
