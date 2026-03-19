## 1. Manifest Schema and Validation Core

- [x] 1.1 Add adapter manifest schema definitions with required fields and compatibility expressions.
- [x] 1.2 Implement manifest parser/validator with deterministic error classification for missing/invalid fields.
- [x] 1.3 Add semver range evaluator supporting pre-release (`-rc`) compatibility checks.
- [x] 1.4 Add unit tests for schema validity, semver boundaries, and failure-classification determinism.

## 2. Runtime Integration Boundary

- [x] 2.1 Integrate manifest loading and validation into adapter activation boundary.
- [x] 2.2 Implement fail-fast behavior for missing manifest, invalid schema, and compatibility mismatch.
- [x] 2.3 Implement required/optional capability enforcement with deterministic downgrade reason for optional path.
- [x] 2.4 Add integration tests for activation success, mismatch fail-fast, required capability fail-fast, and optional downgrade behavior.

## 3. Scaffold and Conformance Linkage

- [x] 3.1 Extend adapter scaffold generator to emit manifest template for `mcp|model|tool`.
- [x] 3.2 Ensure generated manifest defaults include `baymax_compat`, capability sets, and `conformance_profile`.
- [x] 3.3 Extend adapter conformance harness with manifest-profile alignment checks.
- [x] 3.4 Add regression tests to detect scaffold-manifest and conformance-profile drift.

## 4. Gate Integration and Traceability

- [x] 4.1 Add `scripts/check-adapter-manifest-contract.sh` for offline deterministic manifest validation.
- [x] 4.2 Add `scripts/check-adapter-manifest-contract.ps1` with parity to shell behavior.
- [x] 4.3 Integrate manifest contract checks into `scripts/check-quality-gate.sh`.
- [x] 4.4 Integrate manifest contract checks into `scripts/check-quality-gate.ps1`.
- [x] 4.5 Update `docs/mainline-contract-test-index.md` with manifest contract and gate-path mappings.

## 5. Documentation Alignment

- [x] 5.1 Update `README.md` with manifest contract overview and validation entry commands.
- [x] 5.2 Update `docs/external-adapter-template-index.md` with manifest template guidance.
- [x] 5.3 Update `docs/adapter-migration-mapping.md` with manifest migration notes and compatibility semantics.
- [x] 5.4 Update `docs/development-roadmap.md` with A26 scope and sequencing note.

## 6. Validation

- [x] 6.1 Run `go test ./...`.
- [x] 6.2 Run `go test -race ./...`.
- [x] 6.3 Run `golangci-lint run --config .golangci.yml`.
- [x] 6.4 Run `bash scripts/check-adapter-manifest-contract.sh` and `pwsh -File scripts/check-adapter-manifest-contract.ps1`.
- [x] 6.5 Run `bash scripts/check-quality-gate.sh` and `pwsh -File scripts/check-quality-gate.ps1`.
- [x] 6.6 Run `openspec validate introduce-adapter-manifest-and-runtime-compatibility-contract-a26 --strict`.
