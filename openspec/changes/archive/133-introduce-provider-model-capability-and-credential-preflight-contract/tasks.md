## 1. Contract Baseline and Documentation-First Example

- [x] 1.1 Update `examples/agent-modes/MATRIX.md` and the `adapter-onboarding-manifest-capability` README before implementation with a semantic anchor, runtime path, expected catalog/preflight markers, and rollback notes.
- [x] 1.2 Update roadmap and README status snapshots to mark the archived extension-lifecycle proposal accurately and register this proposal as the active next direction.
- [x] 1.3 Update runtime configuration, module-boundary, diagnostics, and contract-test-index documentation for catalog ownership, redacted credential evidence, and required verification evidence.

## 2. Catalog and Credential Admission Contracts

- [x] 2.1 Add a library-owned immutable provider/model catalog value type with normalized identities, catalog version, context window, sorted/deduplicated capabilities, declared fallback, and deterministic validation errors.
- [x] 2.2 Add redacted host-supplied credential-evidence types for `available`, `missing`, `invalid`, and `unverified` statuses with bounded canonical reason codes.
- [x] 2.3 Implement a shared provider-model admission evaluator that reuses `adapter/capability` required/optional negotiation and independently validates declared fallbacks.
- [x] 2.4 Add focused catalog and evaluator tests covering valid input, unknown models, duplicate descriptors, malformed capabilities, nonpositive context windows, required capability denial, optional fallback, and secret-free evidence.

## 3. Runtime Configuration and Readiness Integration

- [x] 3.1 Add `runtime.provider_catalog.*` configuration with `env > file > default` precedence, disabled-by-default compatibility behavior, startup fail-fast validation, and canonical reason mapping.
- [x] 3.2 Extend configuration hot reload to validate complete catalog candidates and atomically retain the preceding snapshot on invalid reloads.
- [x] 3.3 Project provider-model admission through `runtime/config.Manager.ReadinessPreflight()` using existing `ready|degraded|blocked`, strict-mode escalation, and primary-reason arbitration.
- [x] 3.4 Route Run and Stream through the same admission projection and freeze catalog generation, selected model, capability outcome, credential status, fallback, and reason sequence at admission.
- [x] 3.5 Add runtime/config and execution integration tests for precedence, startup failure, reload rollback, strict/non-strict unverified evidence, no provider action on blocked admission, in-flight generation stability, and Run/Stream parity.

## 4. Diagnostics and Replay Evidence

- [x] 4.1 Add additive nullable/default diagnostics fields for catalog version, normalized provider/model identity, capability outcome, credential status, fallback identity, and canonical reasons.
- [x] 4.2 Extend `observability/event.RuntimeRecorder` as the sole writer for provider-model admission diagnostics and add tests proving redaction and legacy-reader compatibility.
- [x] 4.3 Add versioned, redacted replay fixtures and deterministic parser/replay tests for success, unknown model, required denial, optional fallback, credential missing, credential unverified, reload rollback, idempotency, and Run/Stream parity.

## 5. Contract Gates and Verification

- [x] 5.1 Add shell and PowerShell provider-model admission contract gates covering fixture drift, stable ordering, diagnostics redaction, recorder-only writes, and Run/Stream parity; wire both into `check-quality-gate`.
- [x] 5.2 Add conformance checks that prevent Provider SDK imports in `context/*` and prevent remote discovery, credential material, or provider side effects in denied admission paths.
- [x] 5.3 Run focused package tests, `go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `pwsh -File scripts/check-quality-gate.ps1`, and `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 5.4 Run `openspec validate --all`, verify the agent-mode documentation baseline and example markers, and record any intentionally unexecuted verification with its reason. (The Bash provider-model gate was attempted but cannot instantiate in this Windows sandbox: `Bash/Service/CreateInstance/E_ACCESSDENIED`; the equivalent PowerShell gate and full quality gate passed.)

## Example Impact Assessment

修改示例

The existing `adapter-onboarding-manifest-capability` agent-mode example will
be updated rather than adding a new example pattern.
