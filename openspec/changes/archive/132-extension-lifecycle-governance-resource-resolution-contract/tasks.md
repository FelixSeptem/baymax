## 1. Contract and fixture foundation

- [x] 1.1 Define extension descriptor, resource provenance, activation generation, and lifecycle reason taxonomy in the owning Go packages.
- [x] 1.2 Define deterministic source scopes, precedence ranks, tie-break rules, and bounded conflict metadata.
- [x] 1.3 Add versioned replay fixtures for valid/invalid manifest, digest change, compatibility mismatch, required/optional capability, and equal-precedence conflict cases.
- [x] 1.4 Add Example Impact Assessment baseline: create or update an extension manifest/compatibility example README and document expected markers and rollback notes.

## 2. Deterministic resource resolution

- [x] 2.1 Implement source-scoped discovery with stable ordering independent of filesystem enumeration order.
- [x] 2.2 Implement precedence, deduplication, deterministic tie-break, and conflict classification for project/user/explicit/auto/bundled resources.
- [x] 2.3 Persist bounded version/digest/source provenance in the resolved resource projection without activating rejected candidates.
- [x] 2.4 Add offline resolver tests covering unreadable resources, malformed metadata, digest changes, conflict winners, and side-effect-free rejection.

## 3. Manifest, capability, and admission

- [x] 3.1 Implement extension manifest validation for identity, kind, version, compatibility range, digest, and required/optional capabilities.
- [x] 3.2 Reuse adapter capability negotiation for requested-vs-declared extension capabilities and canonical missing/downgrade reasons.
- [x] 3.3 Connect manifest validation to readiness/policy admission with deterministic ready/degraded/blocked mapping and no activation side effects on deny.
- [x] 3.4 Add positive, negative, boundary, and Run/Stream parity tests for manifest compatibility, capability negotiation, and admission decisions.

## 4. Lifecycle execution and reload isolation

- [x] 4.1 Add bounded extension Hook/tool execution with timeout, resource limit, panic recovery, invalid-result handling, and configured skip/deny/degrade behavior.
- [x] 4.2 Enforce turn snapshot reads and save-point mutation commits for extension lifecycle actions.
- [x] 4.3 Implement activation generations, stale old-instance handling, reload success handoff, and atomic rollback on failed reload.
- [x] 4.4 Verify extension failures cannot rewrite the authoritative Run terminal arbiter or introduce a second recovery state machine.
- [x] 4.5 Add fault-injection and concurrency tests for timeout, panic, finalize failure, reload during in-flight action, and stale event suppression.

## 5. Diagnostics, replay, and gates

- [x] 5.1 Emit discovery, admission, activation, hook, failure, reload, and rollback events through RuntimeRecorder only.
- [x] 5.2 Add additive nullable default diagnostics fields with bounded extension identity, generation, phase, reason, source, and digest projections.
- [x] 5.3 Add replay verification for deterministic ordering, capability/admission outcomes, reload rollback, and Run/Stream equivalence.
- [x] 5.4 Add dedicated shell/PowerShell contract and replay gate scripts and wire them into check-quality-gate.* with stable failure classifications.
- [x] 5.5 Add conformance coverage proving policy, sandbox, allowlist, and egress boundaries cannot be bypassed by extension code.

## 6. Documentation and completion verification

- [x] 6.1 Update runtime module boundaries, runtime config/diagnostics, mainline contract index, and roadmap with the finalized extension contract.
- [x] 6.2 Update agent-mode MATRIX/README or the selected extension example with semantic anchor, runtime path, expected markers, and rollback notes.
- [x] 6.3 Run focused package tests and integration/replay suites, then run `go test ./...` and `go test -race ./...`.
- [x] 6.4 Run `golangci-lint run --config .golangci.yml`, quality gate, docs consistency, and OpenSpec validation; record any environment-blocked command explicitly.
