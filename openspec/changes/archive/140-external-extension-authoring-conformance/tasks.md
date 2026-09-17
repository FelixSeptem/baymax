## 1. Contract and case foundation

- [x] 1.1 Define the versioned `external_extension_authoring.v1` case/result schema with bounded fields, enums, canonical ordering, and unknown-field compatibility; verify malformed cases fail before lifecycle evaluation.
- [x] 1.2 Define stable authoring reason-to-remediation mappings for schema, manifest, capability, admission, lifecycle, reload, policy, sandbox, allowlist, and replay drift; verify every emitted reason has bounded guidance.
- [x] 1.3 Add passing and failing fixture seeds for valid activation, missing metadata, compatibility mismatch, capability downgrade/rejection, digest conflict, and malformed case input; verify fixtures are deterministic and side-effect free.

## 2. Offline conformance harness

- [x] 2.1 Implement canonical case decoding and validation with stable map/candidate ordering and profile/version checks; verify repeated decode produces identical canonical input.
- [x] 2.2 Implement harness adapters that call existing extension resolution, manifest/capability admission, lifecycle execution, and generation manager owners; verify rejected candidates never activate.
- [x] 2.3 Implement bounded conformance result and remediation projection; verify output contains only stable phase/reason/field/expected/actual/guidance fields and excludes source, credentials, reasoning, workspace, and unbounded payloads.
- [x] 2.4 Add controlled fault injection for timeout, panic, invalid result, finalize failure, stale generation, and failed reload; verify skip/deny/degrade, generation isolation, and atomic rollback behavior.

## 3. Contract and integration coverage

- [x] 3.1 Add extension package tests for schema, resolution, capability, admission, lifecycle, and reload boundaries with positive, negative, and edge cases; verify canonical reason and remediation output.
- [x] 3.2 Add Run/Stream parity integration tests and policy/sandbox/allowlist/egress non-bypass cases; verify conformance failures cannot rewrite authoritative terminal outcomes.
- [x] 3.3 Add replay tests for deterministic ordering, idempotency, expected/actual drift classification, and stale/rollback fixtures; verify expected fixtures are never silently rewritten.

## 4. Gates and documentation

- [x] 4.1 Add dedicated POSIX and PowerShell conformance/replay gate scripts using one semantic profile; verify both wrappers run equivalent focused tests and classify failures identically.
- [x] 4.2 Wire the new gates into `check-quality-gate.sh/.ps1` and document required-check candidates; verify quality-gate failure labels include a stable conformance classification.
- [x] 4.3 Update `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, and `docs/development-roadmap.md` with owner boundaries, fixture/replay/gate mapping, rollback notes, and active-change status; verify docs consistency passes.
- [x] 4.4 Record Example Impact Assessment as `无需示例变更（附理由）` and verify no `examples/agent-modes` implementation or marker changes are introduced.

## 5. Verification and handoff

- [x] 5.1 Run focused extension and integration/replay tests plus `go test ./...`; verify all pass and capture any environment-blocked command explicitly.
- [x] 5.2 Run `go test -race ./...` and `golangci-lint run --config .golangci.yml`; both pass with MinGW GCC configured via `CC=D:\\tools\\mingw64\\bin\\gcc.exe` (race full suite completed successfully).
- [x] 5.3 Run `pwsh -File scripts/check-quality-gate.ps1`, `pwsh -File scripts/check-docs-consistency.ps1`, and `openspec validate --all`; all required checks pass. The complete 67-step quality gate passed with explicit A64 changed files, the user-authorized narrow waiver `BAYMAX_DIAGNOSTICS_QUERY_BENCH_ENABLED=false`, repository-local cache/APPDATA directories, and MinGW configured through `CC=D:\\tools\\mingw64\\bin\\gcc.exe`. `google.golang.org/grpc` was upgraded to `v1.83.1`, and strict `govulncheck` reports no called vulnerabilities. Docs consistency and OpenSpec validation also pass.
