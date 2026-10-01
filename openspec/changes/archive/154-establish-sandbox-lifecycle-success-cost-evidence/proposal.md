## Why

Baymax already defines sandbox execution, backend/profile conformance, egress, readiness, rollout health, and Run/Stream parity, but it cannot yet compare a bounded sandbox lifecycle against the cost of a completed task. In particular, existing facts do not produce one deterministic, replayable view of acquisition, cold launch, reuse or resume, execution, retry, release or recovery, and the final successful outcome.

This is the right point to establish the evidence layer before any platform-driver or runtime change. Review of Anthropic's public `sandbox-runtime` at `5d196e0` reinforces the need to distinguish platform-specific setup and cleanup from command-level outcomes, to associate violations with an opaque invocation reference, and to keep filesystem-policy replacement separate from live network-policy updates. Baymax will reuse those boundary lessons without importing its backend, process manager, proxy, account, or OS-specific implementation.

## What Changes

- Add a versioned, bounded, offline `sandbox_lifecycle_success_cost_evidence.v1` contract that derives a canonical lifecycle projection from host-supplied sandbox facts.
- Define deterministic lifecycle phases for acquire, launch, execute, retry, release, recover, and terminal outcome; distinguish cold launch from per-session reuse or recovery without creating a sandbox session owner.
- Define success-cost summaries from bounded duration and resource buckets only, plus three evidence verdicts: `within-baseline`, `lifecycle-gap-confirmed`, and `insufficient-evidence`.
- Add fixtures and diagnostics replay evaluation for valid lifecycle evidence, launch/recovery/cleanup gaps, missing or contradictory facts, baseline drift, repeated replay, unknown-field compatibility, and Run/Stream parity.
- Add a dedicated shell/PowerShell contract gate and contribution boundary tests proving that the change introduces no executor/backend, network proxy, filesystem-policy, credential, raw command/output, persistent session store, or new runtime configuration path.
- Document the evidence-only boundary, Anthropic reference assessment, rollback point, and Example Impact Assessment in the roadmap, sandbox diagnostics documentation, and mainline contract-test index.

## Capabilities

### New Capabilities

- `sandbox-lifecycle-success-cost-evidence`: defines bounded, reference-only lifecycle and successful-task cost evidence for existing sandbox facts.

### Modified Capabilities

- `diagnostics-replay-tooling`: adds versioned sandbox lifecycle-success-cost fixture parsing, deterministic drift classification, and backward-compatible replay behavior.
- `go-quality-gate`: adds the sandbox lifecycle-success-cost contract gate to the repository quality-gate contract while preserving shell/PowerShell parity.

## Impact

- Affected code: a pure offline evidence evaluator under `tool/diagnosticsreplay`, sandbox conformance/test-support facts, contribution boundary tests, versioned fixtures, and dedicated shell/PowerShell gate scripts.
- Affected documentation: `docs/development-roadmap.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, `README.md`, and the OpenSpec archive path on completion.
- Compatibility: no runtime model or sandbox execution behavior changes; no `security.sandbox.*` configuration changes; no new diagnostics persistence fields; no provider SDK, container runtime, bubblewrap, Seatbelt, WFP, proxy, service account, or credential dependency.
- Privacy: evidence accepts only opaque invocation/session references, enum states, catalog-like backend/profile identity, bounded duration/resource buckets, retry ordinal, canonical reason codes, and terminal classification. It rejects command text, arguments, environment, workdir, mounts, paths, endpoint, credential material, raw stdout/stderr, raw violation payloads, prompt, reasoning, and unbounded bodies.
- Rollback: remove the evaluator, fixtures, replay handler, boundary/gate checks, documentation, and OpenSpec capability. No persistent state, configuration migration, or sandbox backend cleanup is required.

## Example Impact Assessment

无需示例变更（附理由）：本 change 仅比较宿主或测试支持层提供的既有 sandbox lifecycle facts，不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或 sandbox executor 行为。若后续 confirmed gap 需要改变实际 sandbox session、平台 driver 或执行策略，必须以新 change 先更新 `MATRIX.md` 与受影响模式 README。
