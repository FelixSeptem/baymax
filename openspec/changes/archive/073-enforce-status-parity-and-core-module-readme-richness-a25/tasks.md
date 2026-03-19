## 1. Status Parity Governance Checks

- [x] 1.1 Implement status-parity validator comparing `openspec list --json` and archive index against roadmap/README progress statements.
- [x] 1.2 Add deterministic failure classification for status conflicts (active-vs-archived mismatch, stale snapshot mismatch).
- [x] 1.3 Add contributioncheck tests for status parity conflict detection and success path.

## 2. Core Module README Richness Baseline

- [x] 2.1 Define required section baseline and explicit N/A marker policy for covered core module README files.
- [x] 2.2 Update covered core module READMEs with required enriched sections and actionable module-specific content.
- [x] 2.3 Add contributioncheck tests validating required sections across covered module README list.
- [x] 2.4 Ensure root `README.md` keeps discoverable links to all covered module README files.

## 3. Gate Integration and Traceability

- [x] 3.1 Integrate status-parity and module-readme-richness checks into `scripts/check-docs-consistency.sh`.
- [x] 3.2 Integrate equivalent checks into `scripts/check-docs-consistency.ps1`.
- [x] 3.3 Ensure `scripts/check-quality-gate.sh` continues to block on docs consistency failures.
- [x] 3.4 Ensure `scripts/check-quality-gate.ps1` continues to block on docs consistency failures.
- [x] 3.5 Update `docs/mainline-contract-test-index.md` with status parity and module README gate mappings.

## 4. Documentation Alignment

- [x] 4.1 Update `docs/development-roadmap.md` to keep progress snapshots aligned with current OpenSpec status.
- [x] 4.2 Update root `README.md` milestone snapshot wording to match OpenSpec status and roadmap.
- [x] 4.3 Add brief maintainer note describing how to update snapshots without breaking parity checks.

## 5. Validation

- [x] 5.1 Run `go test ./tool/contributioncheck`.
- [x] 5.2 Run `bash scripts/check-docs-consistency.sh`.
- [x] 5.3 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 5.4 Run `bash scripts/check-quality-gate.sh`.
- [x] 5.5 Run `pwsh -File scripts/check-quality-gate.ps1`.
- [x] 5.6 Run `openspec validate enforce-status-parity-and-core-module-readme-richness-a25 --strict`.
