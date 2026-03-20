## 1. Contract Profile Versioning Core

- [x] 1.1 Add `contract_profile_version` model and parser with recognized profile set (initially `v1alpha1`).
- [x] 1.2 Implement runtime profile compatibility window check with default `current + previous`.
- [x] 1.3 Add deterministic error classification for unknown profile and out-of-window profile.

## 2. Manifest and Negotiation Integration

- [x] 2.1 Extend manifest validation to require `contract_profile_version`.
- [x] 2.2 Wire profile version through negotiation pipeline and taxonomy output path.
- [x] 2.3 Add unit/integration tests for manifest+profile and negotiation+profile combined flows.

## 3. Replay Baseline and Fixtures

- [x] 3.1 Define versioned fixture layout for adapter contract replay by profile version.
- [x] 3.2 Add baseline fixtures for manifest compatibility outcomes.
- [x] 3.3 Add baseline fixtures for negotiation/fallback and reason taxonomy outputs.
- [x] 3.4 Add replay test harness to compare runtime outputs with fixtures deterministically.

## 4. Gate Integration

- [x] 4.1 Add `scripts/check-adapter-contract-replay.sh` for offline deterministic replay validation.
- [x] 4.2 Add `scripts/check-adapter-contract-replay.ps1` with parity to shell behavior.
- [x] 4.3 Integrate replay check into `scripts/check-quality-gate.sh` as blocking step.
- [x] 4.4 Integrate replay check into `scripts/check-quality-gate.ps1` as blocking step.
- [x] 4.5 Ensure replay drift produces fail-fast non-zero exit semantics.

## 5. Documentation and Traceability

- [x] 5.1 Update `README.md` with contract profile version and replay gate overview.
- [x] 5.2 Update `docs/external-adapter-template-index.md` and `docs/adapter-migration-mapping.md` with profile upgrade guidance.
- [x] 5.3 Update `docs/mainline-contract-test-index.md` with A28 replay contract/gate mappings.
- [x] 5.4 Update `docs/development-roadmap.md` with A28 scope and sequencing note.

## 6. Validation

- [x] 6.1 Run `go test ./...`.
- [x] 6.2 Run `go test -race ./...`.
- [x] 6.3 Run `golangci-lint run --config .golangci.yml`.
- [x] 6.4 Run `bash scripts/check-adapter-contract-replay.sh` and `pwsh -File scripts/check-adapter-contract-replay.ps1`.
- [x] 6.5 Run `bash scripts/check-quality-gate.sh` and `pwsh -File scripts/check-quality-gate.ps1`.
- [x] 6.6 Run `openspec validate introduce-adapter-contract-profile-versioning-and-replay-gate-a28 --strict`.

