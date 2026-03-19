## 1. Conformance Harness Skeleton

- [ ] 1.1 Create adapter conformance harness package and shared assertion helpers.
- [ ] 1.2 Define minimum conformance matrix structure for MCP, Model, and Tool categories.
- [ ] 1.3 Implement deterministic fixture/stub layer for offline execution.

## 2. Semantic Contract Cases

- [ ] 2.1 Add MCP conformance cases for request/response normalization and fail-fast boundary checks.
- [ ] 2.2 Add Model conformance cases for Run/Stream equivalence, error-layer normalization, and optional capability downgrade behavior.
- [ ] 2.3 Add Tool conformance cases for invocation contract, error semantics, and fail-fast on invalid mandatory input.
- [ ] 2.4 Add shared checks for reason taxonomy normalization and deterministic failure classification.

## 3. A21 Template Linkage

- [ ] 3.1 Map A21 MCP template path to conformance cases and add trace comments.
- [ ] 3.2 Map A21 Model template path to conformance cases and add trace comments.
- [ ] 3.3 Map A21 Tool template path to conformance cases and add trace comments.
- [ ] 3.4 Add regression checks to fail when template guidance drifts from conformance expectations.

## 4. Gate and Script Integration

- [ ] 4.1 Add `scripts/check-adapter-conformance.sh` for offline deterministic conformance execution.
- [ ] 4.2 Add `scripts/check-adapter-conformance.ps1` with parity to shell behavior.
- [ ] 4.3 Integrate adapter conformance scripts into `scripts/check-quality-gate.sh`.
- [ ] 4.4 Integrate adapter conformance scripts into `scripts/check-quality-gate.ps1`.
- [ ] 4.5 Ensure gate exits fail-fast and non-zero on any conformance mismatch.

## 5. Documentation and Traceability

- [ ] 5.1 Update `docs/mainline-contract-test-index.md` with adapter conformance rows and gate path mapping.
- [ ] 5.2 Update `README.md` and A21 onboarding docs with conformance execution entry.
- [ ] 5.3 Update `docs/development-roadmap.md` with A22 scope and sequencing note.

## 6. Validation

- [ ] 6.1 Run `go test ./...`.
- [ ] 6.2 Run `go test -race ./...`.
- [ ] 6.3 Run `golangci-lint run --config .golangci.yml`.
- [ ] 6.4 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [ ] 6.5 Run `bash scripts/check-adapter-conformance.sh` and `pwsh -File scripts/check-adapter-conformance.ps1`.
- [ ] 6.6 Run `bash scripts/check-quality-gate.sh` and `pwsh -File scripts/check-quality-gate.ps1`.
- [ ] 6.7 Run `openspec validate introduce-external-adapter-conformance-harness-and-gate-a22 --strict`.

