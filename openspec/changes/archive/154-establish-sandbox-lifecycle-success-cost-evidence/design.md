## Context

See `proposal.md` for motivation. Existing sandbox contracts already own execution policy, `host|sandbox|deny` resolution, capability negotiation, `per_call|per_session` semantics, egress, readiness, rollout health, and Run/Stream behavior. `integration/sandboxconformance` has deterministic test support for acquire and close behavior, while `tool/diagnosticsreplay` owns offline fixture normalization and drift validation. Neither is a general lifecycle cost-evidence owner.

Anthropic's public `sandbox-runtime` at revision `5d196e0` is a boundary reference only. Its platform-specific Seatbelt, bubblewrap, WFP, proxy, account, and cleanup mechanisms are host-executor concerns outside this repository change. Its useful lessons are a clear lifecycle taxonomy, opaque command correlation, fail-closed handling for invalid configuration, and explicit distinction between live network policy updates and filesystem-policy reset requirements.

## Goals / Non-Goals

**Goals:**

- Add a pure, bounded normalizer and evaluator for `sandbox_lifecycle_success_cost_evidence.v1` under the existing offline replay/tooling ownership boundary.
- Make cold launch, per-session reuse, recovery or resume, terminal success, and bounded successful-task cost auditable from supplied facts.
- Provide fixtures, replay drift classes, contribution boundaries, and shell/PowerShell gate parity that prove the evaluator remains offline and reference-only.
- Produce a stable evidence verdict that can justify, but never perform, a later sandbox runtime or platform proposal.

**Non-Goals:**

- No new sandbox executor, backend, OS driver, proxy, service account, credential flow, filesystem policy, or egress implementation.
- No `security.sandbox.*` configuration change, new runtime state, persistent diagnostic field, host probe, or platform privilege requirement.
- No production session registry or lifecycle owner; `integration/sandboxconformance` remains test support only.
- No admission, readiness, model routing, terminal-decision, or `examples/agent-modes` behavior change.

## Decisions

### 1. Use an evidence projection instead of an execution lifecycle owner

The implementation will accept a host-supplied record composed of opaque invocation/session references, backend/profile identity, session mode, phase events, bounded duration/resource buckets, retry ordinal, stable reason code, terminal classification, and optional Run/Stream snapshots. It will normalize those facts and return an immutable canonical projection, digest, verdict, reasons, and optional unit-success-cost summary.

This preserves the existing runtime and host-executor owners and prevents a second session state machine. A global session registry or direct integration with `integration/sandboxconformance` was rejected because it would introduce production lifecycle ownership and mutable state without evidence that it is needed.

### 2. Make continuity explicit and cost terminal-success-only

The evaluator will process the fixed phase taxonomy `acquire`, `launch`, `execute`, `retry`, `release`, `recover`, and terminal outcome. Normalized facts explicitly identify cold launch, valid per-session reuse, and recovery or resume; invalid sequence claims classify deterministically rather than being repaired.

Only complete, internally consistent terminal-success evidence can produce a unit-success-cost summary. Cost is represented by cataloged duration/resource buckets, never raw consumption or runtime observation. This avoids treating failures or missing data as cost savings or product gaps.

An alternative that aggregates all task attempts was rejected because it can obscure retry, recovery, and failure effects and cannot answer the roadmap's unit successful-task question safely.

### 3. Keep verdicts evidence-oriented and offline

The evaluator returns exactly one of `within-baseline`, `lifecycle-gap-confirmed`, or `insufficient-evidence`. It compares supplied normalized buckets with supplied baseline expectations only. It neither probes a host nor modifies scheduling, admission, rollout, or execution.

`lifecycle-gap-confirmed` requires complete allowed facts that demonstrate the declared representational gap. Missing or contradictory facts remain `insufficient-evidence`, so ambiguity cannot accidentally authorize a sandbox platform build.

### 4. Put replay integration at the tooling boundary

The pure evaluator and fixture adapter will live under the existing `tool/diagnosticsreplay` ownership area, with focused fixtures and tests. `tool/contributioncheck` will hold static boundary assertions, and dedicated shell/PowerShell scripts will invoke the same offline suites. The scripts will never launch an OS sandbox; their purpose is contract conformance, not platform verification.

Fixture v1 output will support a canonical digest, verdict, bounded summaries, stable reason ordering, drift classes, unknown-field-safe compatibility, and repeated-replay idempotence. Existing historical fixture parsing stays unchanged except for dispatching the new v1 type.

### 5. Treat Anthropic findings as constraints, not dependencies

The evidence taxonomy reflects cleanup and recovery concerns visible in Anthropic's runtime, while opaque invocation correlation follows its separation of `commandId` from command text. The policy-update distinction becomes an audit constraint: this change does not claim that a filesystem policy can be replaced for a live process because it adds no policy feature at all. Windows readiness, ACL recovery, WFP, and all platform driver details remain external host/executor behavior.

Copying `sandbox-runtime`, importing TypeScript process-management patterns, or introducing its platform helpers was rejected because Baymax has no confirmed runtime/platform gap yet and those choices would violate the evidence-only scope.

## Risks / Trade-offs

- **[Risk] A minimal fact set may often yield `insufficient-evidence`.** -> This is intentional fail-closed behavior; fixtures establish the minimum complete record and later additions require a versioned contract change.
- **[Risk] Bounded buckets conceal fine-grained cost variation.** -> The contract answers a stable baseline question first; raw measurements and live profiling remain a separate, explicitly scoped proposal.
- **[Risk] Consumers may mistake `lifecycle-gap-confirmed` for authorization to change runtime behavior.** -> Output is reference-only, gates forbid side effects, and any executor/config/platform change requires a new OpenSpec change.
- **[Risk] Lifecycle taxonomy diverges from existing sandbox semantics.** -> Reuse existing session-mode and terminal vocabulary where applicable, cover invalid continuity with fixtures, and preserve Run/Stream parity.
- **[Risk] A fixture leaks sensitive operational data.** -> Normalization rejects prohibited fields before projection; privacy drift fixtures and contribution checks block regression.

## Migration Plan

1. Add data normalization and pure evaluator tests before integrating fixtures.
2. Add v1 fixtures, replay dispatch, drift classifications, idempotence, and mixed-fixture compatibility tests.
3. Add contribution boundary tests, contract scripts, documentation, and quality-gate registration.
4. Run focused and repository gates. The evidence result may inform a future change only after it is independently reviewed.

Rollback removes the evaluator, fixtures, replay dispatch, boundary tests, scripts, quality-gate registration, and documentation together. There is no runtime migration, persisted record, configuration value, platform resource, or sandbox backend to clean up.

## Example Impact Assessment

无需示例变更（附理由）：本设计仅处理离线、宿主或测试支持层提供的 lifecycle evidence，不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或实际 sandbox executor。任何将 confirmed gap 转为 session、platform driver 或执行策略变化的后续 change，必须先完成 `MATRIX.md` 与对应模式 README 的文档基线。
