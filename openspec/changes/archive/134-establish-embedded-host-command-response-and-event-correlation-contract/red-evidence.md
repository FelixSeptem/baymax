# Pre-implementation RED evidence

## Baseline and method

The implementation is still uncommitted on the working tree. The repository `HEAD` used for the pre-implementation baseline is:

```text
027e904035b19abe0ba1d812ab3c2cb242f36854
```

That commit predates the Embedded Host implementation and does not contain `host/`, `host/jsonl/`, `cmd/host-jsonl/`, `core/types/host_contract.go`, `core/runner/control.go`, or `tool/diagnosticsreplay/host_transcript.go`.

For the RED replay, an isolated copy of that commit was created under `.tmp/host-red-baseline`. Only the newly added contract tests and the versioned transcript fixture were copied into that isolated tree. No implementation source was copied. Each focused suite was then run with an isolated `GOCACHE`.

## Unified focused RED matrix

| Contract area | Focused command | Baseline result | Missing implementation signal |
| --- | --- | --- | --- |
| Envelope validation and command DTOs | `go test ./core/types -run 'Host|Protocol' -count=1` | non-zero | `undefined: HostCommandEnvelope`, `HostProtocolVersionV1`, `HostCommandKindRunStart` |
| Active control, Realtime ingress, duplicate/late and Run/Stream control parity | `go test ./core/runner -run 'Retry|ActiveRun|Realtime|Host' -count=1` | non-zero | missing `RetryRun`, `ActiveRunControl`, `WithActiveRunControlLimit`, registry methods |
| Command/event separation, terminal races, HITL races, disconnect cleanup, source ownership and delivery | `go test ./host -count=1` | non-zero | missing host DTOs and coordinator-facing implementation (`HostResponseEnvelope`, `HostRunStartAdmission`, `HostRunExecutionMode`, etc.) |
| JSONL framing, serialized output, output failure, EOF and backpressure | `go test ./host/jsonl -count=1` | non-zero | baseline has no non-test `host` implementation files |
| Real subprocess negotiation, stdout purity, framing, pending cleanup and bounded backpressure | `go test ./cmd/host-jsonl -run 'Conformance|Executable' -count=1` | non-zero | baseline has no non-test `host/jsonl` implementation files |
| Versioned transcript replay, duplicate/HITL/disconnect/terminal/framing/output/backpressure cases and Run/Stream parity | `go test ./tool/diagnosticsreplay -run 'HostTranscript' -count=1` | non-zero | missing `EvaluateHostTranscriptFixtureJSON`, `HostTranscriptFixtureV1`, and parity drift reason |

The six suites all failed before any implementation source was present. The transcript fixture contains the complete first-profile scenario set: valid flow, rejection, duplicate, HITL, disconnect/recovery, terminal conflict, framing rejection, output failure, backpressure, and Run/Stream parity/idempotency. The focused test names cover envelope validation, command/event separation, duplicate/late operations, terminal races, HITL response races, Realtime ingress, disconnect cleanup, JSONL framing, backpressure, and Run/Stream parity.

This is a reproducible baseline reconstruction from the actual pre-implementation commit, not a fabricated post hoc passing assertion. The raw isolated working tree is disposable; the commit, injected test set, commands, and failure signatures above are sufficient to replay the RED check.
