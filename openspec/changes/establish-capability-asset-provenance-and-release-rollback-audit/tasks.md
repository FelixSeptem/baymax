## 1. Contract and canonical projection

- [x] 1.1 Define the versioned `capability_asset_provenance.v1` schema, bounded enums, nullable/default rules, and privacy limits; verify the schema covers every ADDED requirement and rejects missing required identity, kind, owner, or scope fields with `go test ./tool/diagnosticsreplay -run CapabilityAssetProvenance`.
- [x] 1.2 Implement deterministic normalization, reference deduplication/sorting, canonical digest calculation, and unknown additive-field handling; verify equivalent input order produces identical normalized output and digest with the focused replay tests.
- [x] 1.3 Implement bounded dependency/consumer reference validation and scope checks; verify malformed, body-bearing, cross-scope, and over-limit references produce stable classifications without accepted output with the focused negative tests.

## 2. Audit findings and impact projection

- [x] 2.1 Implement provenance comparison for identity, version, digest, owner, scope, dependency, and consumer drift; verify missing evidence, malformed metadata, declared drift, and conflicting duplicate identity remain distinct findings.
- [x] 2.2 Implement reference-based withdrawal impact calculation with complete/incomplete/conflicting evidence states; verify sorted bounded impact sets and stable impact digests without mutating owner state.
- [x] 2.3 Implement advisory replacement compatibility evaluation; verify compatible and incompatible replacements produce bounded findings and never activate, withdraw, or roll back assets.

## 3. Fixtures, replay, and parity

- [x] 3.1 Add versioned offline fixtures for valid assets, nullable historical fields, duplicate references, drift, scope violations, dependency conflicts, consumer conflicts, withdrawal impact, and replacement compatibility; verify fixtures contain no credentials, raw bodies, reasoning, or workspace content.
- [x] 3.2 Add deterministic replay normalization and finding comparison with idempotency checks; verify repeated replay returns identical projection, finding order, evidence digests, and completeness semantics.
- [x] 3.3 Add equivalent Run/Stream envelope coverage and boundary tests; verify both entry points produce equivalent projection, findings, and impact completeness without provider or network calls.

## 4. Gates, documentation, and governance

- [ ] 4.1 Add focused contract/replay gate scripts with shell and PowerShell parity; verify both wrappers fail on schema, digest, finding, or privacy drift and pass offline on the checked-in fixtures. PowerShell wrapper passes; shell wrapper is present and equivalent but cannot execute in this Windows environment because bash/WSL is unavailable (`CreateProcessCommon: execvpe(/bin/bash) failed`).
- [x] 4.2 Wire the focused gate into the existing quality-gate and contract-test index without changing `RuntimeRecorder`, runtime configuration, or authoritative owner semantics; verify the gate is discoverable and bounded.
- [x] 4.3 Update roadmap, capability documentation, and relevant module boundary notes with the reference-only ownership and rollback limits; verify docs consistency reports no roadmap-status drift and Example Impact Assessment remains `无需示例变更（附理由）`.

## 5. Verification and delivery evidence

- [x] 5.1 Run focused package tests, replay/contract gates, and `go test -race` for affected packages; verify positive, negative, and boundary scenarios are covered. PowerShell gate, focused package tests, and race tests passed; shell wrapper execution remains environment-blocked by Windows bash permission failure.
- [ ] 5.2 Run repository-required `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, and `pwsh -File scripts/check-docs-consistency.ps1`; record successful output and leave unrelated workspace artifacts untouched. `go test -race ./...`, golangci-lint, and docs consistency passed; plain `go test ./...` is blocked by pre-existing `cmd/host-jsonl` conformance/cleanup failures; quality gate is blocked by pre-existing agent-mode `shared-wrapper-detected` failure in `examples/agent-modes/rag-hybrid-retrieval/minimal/main.go`.
- [x] 5.3 Review the final diff for architecture-boundary compliance, additive/null/default compatibility, privacy limits, and absence of runtime release/rollback side effects; verify OpenSpec validation passes before archive consideration.
