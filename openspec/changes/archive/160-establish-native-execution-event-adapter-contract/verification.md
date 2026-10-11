# Implementation verification

Change: `establish-native-execution-event-adapter-contract`. Implementation branch: `feat/native-execution-event-adapter`, based on `553e076`. Verification runs on Windows PowerShell and Git Bash; Git Bash verifies the shell entrypoint on Windows, not a Linux/macOS runtime.

## Semantic and documentation audit

| Public semantic | Maintained documentation | Verification evidence |
| --- | --- | --- |
| Additive `native_execution_event.v1`, immutable pure projection, explicit source binding | `core/types/README.md`, `docs/architecture/data-event-flow.md`, `docs/runtime-module-boundaries.md` | Projection baseline, source/correlation and detached snapshot tests; ownership AST mutation tests |
| Direct callback mapping references, sequence zero, no dedup/global order/history/cursor | `host/eventadapter/README.md`, `docs/guides/integration.md`, `docs/guides/best-practices.md` | Shared TraceID/fallback identity/repeated callback tests, unsupported capability admission and fake-fact replay |
| Source-owned Realtime binding only; overlap/gap/expired/correlation retained | `host/eventadapter/README.md`, `core/types/README.md` | Realtime binding/metadata tests and versioned fixtures |
| Fixed host reference allowlist, strict JSON types including null rejection, default 1024-byte scalar/16 citations/8-KiB references/64-KiB event | `host/eventadapter/README.md`, `docs/runtime-config-diagnostics.md` | Strict reference types, exact byte/item limits and overflow tests |
| Bounded snapshot, number precision, custom JSON/Text marshalers rejected before invocation, including addressable pointer methods | `host/eventadapter/README.md`, `core/types/README.md` | Mutation/large-integer/cyclic/non-JSON tests and custom serialization regressions |
| Independent observer context; synchronous context-aware sink; default one-second timeout; no queue/retry/worker | `host/eventadapter/README.md`, `docs/guides/integration.md`, `docs/runtime-config-diagnostics.md` | Observer deadline/override/context/concurrency/reentry/Close tests and ownership gate |
| First failure stops delivery; bounded status; write-then-error is unconfirmed; no execution control | `host/eventadapter/README.md`, `docs/guides/best-practices.md` | First reason/reference/counter bounds, receipt ambiguity and real-runner source independence |
| Pause is nonterminal; confirm/deny/timeout/duplicate resume cardinality owned by source | `docs/guides/integration.md`, `host/eventadapter/README.md` | Real Run/Stream dynamic action tests; pause-as-terminal negative fixture |
| RuntimeRecorder remains independent and sole diagnostic writer; no runtime keys or QueryRuns schema changes | `observability/README.md`, `docs/runtime-config-diagnostics.md`, `docs/runtime-module-boundaries.md` | Recorder fan-out integration and ownership gate |
| Offline parity normalizes only schema-owned generated values and correlated fragment-only deltas | `docs/mainline-contract-test-index.md`, `host/eventadapter/README.md` | Lifecycle/resolution/identity/correlation/failure/receipt drift plus semantic delta, business-field and fragment-profile counterexamples |
| Host owns persistence, authorization, business commits and transport; migration/rollback is opt-in wiring | `docs/guides/integration.md`, `docs/guides/best-practices.md`, `examples/native-execution-events/README.md`, `README.md`, roadmap | Real fake-model/tool example markers, docs consistency and contract gates |

No changes to `examples/agent-modes` are part of this implementation. The example README baseline preceded its code. No live provider, application write or external service is used by the new conformance/example suites. Inline review retained the existing protocol mapper and source execution owners.

## Verification record

Resource controls for full correctness checks: `GOMAXPROCS=2`, `GOFLAGS=-p=1`. The initial quality gate used serial steps; reruns limit independent contract steps to two (`BAYMAX_QUALITY_GATE_PARALLELISM=2`). Performance baselines use the default CPU setting (16 on this host), with `GOFLAGS=-p=1` retained to bound compilation. No checks are disabled, and no baseline or degradation threshold is changed.

| Command | Result |
| --- | --- |
| Affected projection/observer/integration/replay/ownership tests and offline example | Passed in both final dedicated gates |
| `go test ./...` | Passed on final source in quality step 71 (298.99 seconds) |
| `go test -race ./...` | Passed on final source in quality step 72 (722.56 seconds) |
| `golangci-lint run --config .golangci.yml` | Passed on final source: `0 issues`, quality step 73 (64.90 seconds) |
| PowerShell and Git Bash native execution contract gates | Passed after final reference/custom-marshaler corrections; Git Bash required escalation for signal-pipe access |
| `pwsh -File scripts/check-quality-gate.ps1` | Initial serial run passed steps 1–70, then exhausted its 2400-second total budget during full `go test ./...` (remaining step budget 46 seconds), with no test failure assertion. Second run passed steps 1–74, including full tests/race/lint/semantic checks, and stopped at performance step 75 under `GOMAXPROCS=2`. Final default-CPU rerun passed all native contract, full test, race, lint, semantic, performance and full-chain steps; strict `govulncheck` reported existing Go/net vulnerabilities (exit 3). A warn-mode rerun reached the same checks but reproduced the existing Windows `cmd/host-jsonl` child-process cleanup lock during full `go test ./...`; the affected package passed independently with a writable repository cache. |
| Diagnostics query performance regression | Default-CPU reruns passed unchanged thresholds: mailbox median 0.97 ms / p95 2.52 ms, then 0.90 ms / p95 2.36 ms. One intervening sample failed the existing sandbox-enriched recorder ns/op threshold (266722 ns/op); subsequent median was 175924 ns/op and passed. No adapter wiring is used by these existing benchmarks |
| `pwsh -File scripts/check-full-chain-example-smoke.ps1` | Passed: `FULL_CHAIN_SUCCESS`, Run/Stream aligned, async/delayed/recovery checks passed |
| `pwsh -File scripts/check-docs-consistency.ps1` | Passed after final documentation cleanup |
| `openspec validate establish-native-execution-event-adapter-contract --strict` | Passed after final review |
| `git diff --check` | Passed after final cleanup |

Earlier concurrent full checks exhausted Windows memory/thread resources: race compilation and lint subprocesses failed allocation; fixed-delay legacy configuration tests and JSONL frame-count tests also failed. Both legacy packages passed under reduced concurrency without source changes. A sandbox cache access denial required escalated Go verification.

The second full gate's mailbox benchmark used two CPUs to reduce correctness-test resource pressure, unlike the existing baseline's default CPU configuration. Its mailbox median was 1.84 ms and p95 6.40 ms (48 allocations), exceeding the unchanged 12%/15% thresholds. Restoring default CPU availability without changing source passed the diagnostics suite twice; the intervening recorder sample failure is retained above rather than discarded from the verification record. Final quality evidence is complete for this change; the remaining warn-mode full-gate failure is an environment-level Windows cleanup lock in `cmd/host-jsonl`, not a native execution event failure. Strict vulnerability findings remain an existing dependency baseline issue and are recorded rather than suppressed.

Extra `openspec validate --all --strict` reported 24 passing and 112 failing legacy items (136 total), principally existing Purpose placeholders. The new change passed strict validation; unrelated legacy specs are not rewritten by this change.

## Risks and rollback

Arbitrary Go sinks must cooperate with context deadlines; the library cannot forcibly stop an uncooperative callback. Direct mapped IDs remain nonunique references, so hosts must not deduplicate callbacks with them. Projection has bounded copy/serialization cost; the 4-KiB benchmark measured approximately 57.6 microseconds, 28 KiB and 82 allocations on the local Windows AMD 7840HS, with no performance threshold claimed. Missing or failed terminal delivery is observation incompleteness, not a business execution failure or permission to retry writes.

Rollback removes the new observer wiring and restores the host's explicit EventHandler path. No database/configuration migration occurs, source states remain authoritative, and external effects or host records are not undone. Native runner durable history and execution resume are outside this change.
