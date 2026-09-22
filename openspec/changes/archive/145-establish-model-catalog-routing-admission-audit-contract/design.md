## Context

See `proposal.md` for motivation. The current repository already owns provider/model descriptor normalization and immutable catalog publication in `model/catalog` and `runtime/config`; capability negotiation remains in `adapter/capability`; readiness and admission remain in `runtime/config`; diagnostic writes remain owned by `observability/event.RuntimeRecorder`. Existing behavior evaluates an exact provider/model identity and may use a declared fallback. The missing evidence is whether a host needs deterministic selection among multiple supplied candidates and whether that selection can be represented without a new control plane.

The design must preserve `env > file > default`, fail-fast validation with atomic reload rollback, additive/null/default diagnostic compatibility, Run/Stream semantic equivalence, and the prohibition on provider SDK dependencies in shared/runtime packages.

## Goals / Non-Goals

**Goals:**

- Define a bounded, versioned, provider-neutral audit contract for host-supplied catalog candidates.
- Make candidate normalization, capability filtering, credential evidence, fallback, skip reasons, selected identity, catalog generation, and Run/Stream parity replayable and drift-detectable.
- Add a pure deterministic candidate resolver only when the audit demonstrates that exact-identity admission cannot express a real host requirement.
- Keep all resolver inputs host-supplied and side-effect free; preserve existing readiness, admission, credential, fallback, and terminal owners.
- Provide positive, negative, boundary, privacy, historical-compatibility, replay-idempotency, and shell/PowerShell gate coverage.

**Non-Goals:**

- Remote or background model discovery, refresh workers, network calls, credential probes, credential storage, or endpoint management.
- A global mutable router, a second catalog source of truth, provider-agnostic wire protocol, gateway, or hosted control plane.
- Automatic model switching, prompt/policy/memory/tool mutation, or mid-stream provider switching.
- New runtime configuration keys before fixture evidence proves the existing host-supplied candidate input is insufficient.
- Changes to provider SDK adapters or `context/*` dependencies in the audit phase.

## Decisions

### 1. Audit first, resolver second

The change introduces a new `model-catalog-routing-admission-audit` contract and fixture family. The first implementation stage records whether a host-supplied candidate set can be normalized and evaluated through existing exact-identity admission. A resolver is conditional: it is added only if an audit fixture demonstrates an expressiveness gap. This keeps roadmap trigger evidence separate from implementation assumptions.

The initial audit fixture now demonstrates that two independently admissible,
host-supplied candidates cannot be represented by the existing exact-identity
`Evaluate` path without either selecting by caller order or returning an ambiguous
result. That is the recorded expressiveness gap for this change. The resolver is
therefore enabled as an opt-in pure function, while the existing exact-identity
path remains unchanged.

Alternative considered: immediately add a resolver. Rejected because it would freeze ranking and fallback semantics without a reproduced host need.

### 2. Host-supplied candidate set, no discovery

Candidate identities, catalog generation, descriptor metadata, credential evidence, and optional priority hints are inputs supplied by the host/config snapshot. The contract never discovers candidates or validates credentials itself. Candidate order is normalized into a deterministic sequence; duplicate identities and conflicting priority metadata fail fast rather than using last-write-wins.

Alternative considered: discovery SPI or remote catalog. Rejected by the roadmap and module boundaries; it would introduce lifecycle, network, security, and ownership concerns outside this change.

### 3. Pure resolver with explicit tie-break and admission reuse

If required by audit evidence, `model/catalog` exposes a pure resolver that evaluates candidates using existing capability negotiation and credential/readiness policy. It returns selected identity, ordered skip reasons, capability outcome, credential status, fallback identity, and catalog generation. The resolver is deterministic for equivalent inputs and rejects ambiguous ties. It never invokes a provider, mutates runtime state, or performs I/O.

Alternative considered: put selection in runtime/readiness or provider adapters. Rejected because catalog facts belong to `model/catalog`, while readiness remains an observer/admission owner and provider SDK details must remain provider-owned.

### 4. Bounded replay projection

The fixture records identities, enums, bounded reason codes, capability sets, ordinals, generation identifiers, digest/length summaries, and selected/ skipped correlations. It does not record credentials, endpoints, raw responses, prompts, or unbounded bodies. Replay compares canonical selection and admission facts twice to prove idempotency and checks historical fixtures with absent optional fields.

### 5. Additive diagnostics only

If runtime diagnostics need new facts, they are nullable/default-compatible projections written only through `RuntimeRecorder`. The initial audit prefers fixture-only facts and does not add configuration keys or diagnostic fields unless the audit identifies a concrete gap and the spec is updated first.

## Risks / Trade-offs

- **[Risk] Candidate priority semantics become an accidental public API.** → Keep priority optional, bounded, and reject ambiguous ties; require fixture evidence before enabling resolver behavior.
- **[Risk] Resolver duplicates existing fallback or readiness logic.** → Delegate capability negotiation and strict/degraded mapping to existing owners; resolver only orders candidate facts.
- **[Risk] Host candidate order creates nondeterministic Run/Stream behavior.** → Canonicalize order and compare normalized selection digests for both paths.
- **[Risk] Credential or endpoint data leaks into fixtures/diagnostics.** → Accept only normalized status and bounded reason codes; add negative privacy fixtures and contribution boundary checks.
- **[Risk] Existing exact-identity callers regress.** → Preserve current `Evaluate` behavior and make candidate resolution opt-in to the audit/resolver path; retain historical fixtures unchanged.
- **[Risk] The audit finds no gap, making the change appear under-delivered.** → Treat a proven no-gap result as a valid outcome: deliver the contract, fixtures, replay and gates, and leave the resolver task explicitly conditional/not activated.

## Migration Plan

1. Create the versioned audit fixture and pure normalization/replay contract without changing runtime selection behavior.
2. Run the fixture against existing catalog, capability, credential, fallback, readiness, and Run/Stream paths.
3. If and only if a reproducible expressiveness gap is observed, update the spec delta and implement the pure host-supplied resolver with focused tests.
4. Keep resolver behavior opt-in behind existing admission inputs; invalid catalog candidates continue to fail fast and retain the prior atomic snapshot.
5. Validate focused tests, replay idempotency, contribution boundaries, both platform gates, docs consistency, and full quality gates.
6. Rollback removes the new fixture/replay/gate and any resolver projection; no persistence migration or remote state cleanup is required.

## Example Impact Assessment

无需示例变更（附理由）。本 change 的首阶段是 catalog/routing audit、fixture、replay 和 gate；不改变 `examples/agent-modes` 的配置、runtime path 或 expected markers。任何后续 resolver 行为若改变示例可观察选择，必须先完成 `MATRIX.md` 与受影响模式 README 的文档基线，并将评估改为“修改示例”。
