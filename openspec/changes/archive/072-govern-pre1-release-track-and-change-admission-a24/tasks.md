## 1. Roadmap and Governance Policy Alignment

- [x] 1.1 Update `docs/development-roadmap.md` to explicitly retain `0.x` posture and remove implicit `1.0/prod-ready` commitments.
- [x] 1.2 Align `docs/versioning-and-compatibility.md` wording with roadmap pre-1 governance posture.
- [x] 1.3 Update contributor-facing release snapshot text in `README.md` if wording conflicts with pre-1 posture.

## 2. Proposal Admission Rule Formalization

- [x] 2.1 Define proposal admission checklist fields (`Why now`, risk, rollback, docs impact, validation commands) in governance docs.
- [x] 2.2 Document bounded objective categories for near-term pre-1 proposals and long-term deferral rules.
- [x] 2.3 Ensure long-term platformization topics are explicitly marked non-near-term in roadmap.

## 3. Docs Consistency Gate Extension

- [x] 3.1 Extend `scripts/check-docs-consistency.sh` to validate pre-1 governance stage consistency.
- [x] 3.2 Extend `scripts/check-docs-consistency.ps1` with equivalent semantics and fail-fast behavior.
- [x] 3.3 Add or update contributioncheck tests covering pre-1 stage conflict detection and non-zero failure behavior.

## 4. Quality Gate Integration and Traceability

- [x] 4.1 Ensure docs consistency checks with pre-1 governance assertions remain in `scripts/check-quality-gate.sh`.
- [x] 4.2 Ensure docs consistency checks with pre-1 governance assertions remain in `scripts/check-quality-gate.ps1`.
- [x] 4.3 Update `docs/mainline-contract-test-index.md` to map governance consistency checks and gate paths.

## 5. Validation

- [x] 5.1 Run `go test ./tool/contributioncheck`.
- [x] 5.2 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 5.3 Run `bash scripts/check-docs-consistency.sh`.
- [x] 5.4 Run `pwsh -File scripts/check-quality-gate.ps1`.
- [x] 5.5 Run `openspec validate govern-pre1-release-track-and-change-admission-a24 --strict`.
