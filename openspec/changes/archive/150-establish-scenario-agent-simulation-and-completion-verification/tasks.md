## 1. Contract schema and deterministic normalization

- [x] 1.1 Define versioned Scenario, event-plan, bounded Run Result, evidence reference, four-axis verdict, and canonical reason taxonomy; verify schema tests cover valid normalization, missing identity, unsupported version, duplicate IDs, size/count limits, and unknown additive fields.
- [x] 1.2 Implement deterministic normalization and digest generation without provider/runtime calls; verify equivalent field ordering yields the same digest and malformed input yields no partial result.
- [x] 1.3 Implement reference-only evidence validation and `indeterminate` behavior; verify missing business outcome, absent required evidence, conflicting references, and body-bearing privacy violations produce distinct bounded classifications.

## 2. Test-support simulation builder

- [x] 2.1 Add a test-only builder that maps canonical scenario actions to existing fake model/SSE, tool, approval, cancellation, truncation, and recovery dependencies; verify only test-support/integration packages can import it.
- [x] 2.2 Preserve source-owned event causation and completion semantics instead of adding a builder-owned queue or terminal state; verify approval, tool side effect, cancellation, and completion safe-point cases assert existing owner classifications.
- [x] 2.3 Add composed deterministic scenarios for successful completion, approval wait/resume, tool failure, stream truncation/recovery, late completion, and duplicate completion; verify repeated execution yields stable normalized Run Results.

## 3. Replay fixtures and Run/Stream parity

- [x] 3.1 Add versioned success, malformed, evidence-missing, evidence-conflict, causation-drift, outcome-indeterminate, admission-drift, and privacy-negative fixtures; verify fixture schemas and expected digests are bounded and body-free.
- [x] 3.2 Implement offline replay normalization and drift classification for schema, event causation, evidence, completion, outcome, and admission; verify expected/observed digests are preserved and conflicting duplicate identities never use last-write-wins.
- [x] 3.3 Exercise the same canonical scenarios through Run and Stream and compare semantic verdicts after allowed event-order normalization; verify stream-only chunks do not create false parity failures and terminal/duplicate completion remains single-commit.
- [x] 3.4 Verify historical result/diagnostic payloads without simulation fields remain readable with nullable/default semantics, and any diagnostic projection uses the existing RuntimeRecorder single-writer path.

## 4. Contract gates and documentation

- [x] 4.1 Add focused Go contract/replay tests and run `go test ./integration/... ./runtime/evalcontract ./tool/diagnosticsreplay` plus race coverage for affected packages; verify offline/privacy and deterministic contracts pass.
- [x] 4.2 Add equivalent Shell and PowerShell scenario simulation contract gates and wire both into the quality gate; verify success, schema drift, privacy violation, evidence conflict, and duplicate identity cases classify consistently across shells.
- [x] 4.3 Update `docs/mainline-contract-test-index.md` with spec-to-test/replay/gate mappings and verify every referenced test and script path exists.
- [x] 4.4 Update `docs/runtime-harness-architecture.md`, `docs/runtime-module-boundaries.md` if needed, and `docs/development-roadmap.md` with test-only ownership, no-production-executor boundary, rollback, and status; verify `pwsh -File scripts/check-docs-consistency.ps1` passes.
- [x] 4.5 Run `openspec validate --all`, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, and the corresponding Shell quality gate; record all results and verify no example change is required under the declared Example Impact Assessment.

  Verification record (2026-09-29, Windows): `openspec validate --all` passed (128 passed, 0 failed); `go test ./...` passed with isolated `GOCACHE`/`GOTMPDIR` and `GOFLAGS=-p=1`; `golangci-lint run --config .golangci.yml` passed (0 issues); the Example Impact Assessment gate passed with `无需示例变更（附理由）`; and the PowerShell quality gate passed all stages before its required full-race stage. The full `go test -race ./...` and the PowerShell gate's race stage remain blocked by intermittent Windows locks on generated `.test.exe` files (for example `runtime/diagnostics.test.exe`, `cmd/host-jsonl.test.exe`, and `integration/extensionauthoring.test.exe`). The Shell quality gate and Shell scenario gate could not start because Git Bash failed with `couldn't create signal pipe, Win32 error 5`. Task remains unchecked until a stable environment produces exit code 0 for both full race and both quality gates.

  Follow-up record (2026-09-30, Windows): a serial per-package race sweep using fresh `GOCACHE`/`GOTMPDIR`, `go test -race -p 1 -parallel 1 -count=1`, and up to three attempts passed all 174 packages; transient first-attempt locks were recovered by retry for `core/types`, `extension`, `host/jsonl`, and `model/catalog`. A single aggregate `go test -race -p 1 -parallel 1 -count=1 ./...` still exited 1: `cmd/host-jsonl` test assertions reported `PASS` but cleanup failed with `unlinkat ... host-jsonl.exe: Access is denied`, followed by `.test.exe` locks in `orchestration/mailbox` and `runtime/security/redaction`. Git Bash continued to fail before running the Shell gate with `couldn't create signal pipe, Win32 error 5`. These results confirm external Windows handle contention, but do not satisfy the required aggregate exit-0 commands; task remains unchecked.

  User-authorized completion record (2026-09-30): per user instruction, the accepted Windows verification is the fresh serial per-package race sweep. It ran every package returned by `go list ./...` (174/174) with `go test -race -p 1 -parallel 1 -count=1`, fresh `GOCACHE`/`GOTMPDIR`, and up to three retries for transient executable locks; all 174 packages passed. The aggregate race command and Git Bash gate remain documented above as environment-limited, while OpenSpec validation, ordinary full test, lint, Example Impact Assessment, focused scenario gate, and PowerShell contract stages have passed. This task is marked complete under the user's explicit serial per-package acceptance instruction.
