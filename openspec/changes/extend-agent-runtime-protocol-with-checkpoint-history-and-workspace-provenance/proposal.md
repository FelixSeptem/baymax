## Why

The Agent Runtime Protocol currently exposes a reference-only checkpoint, but hosts cannot determine checkpoint history, branch lineage, replay identity, or which workspace change-set produced a recoverable state. Long-running recovery and cross-session restore therefore require source-specific knowledge and make workspace integrity drift difficult to audit. This change closes that protocol projection gap while preserving snapshot and sandbox ownership.

## Example Impact Assessment

新增示例

Add a documentation-first checkpoint/workspace provenance example with MATRIX and README coverage before executable smoke assertions.

## What Changes

- Add an additive checkpoint history and workspace provenance projection capability.
- Extend `CheckpointRef` with optional parent, branch, history, restore-source, and replay references.
- Add a bounded `WorkspaceProvenance` reference containing change-set identity, before/after integrity references, and producing Run/Step correlation.
- Add pure snapshot projection and validation for lineage continuity, branch conflicts, replay idempotency, schema compatibility, and workspace association/drift.
- Preserve strict/compatible restore behavior and keep snapshot manifest as the sole storage fact source.
- Add deterministic replay fixtures, drift classes, Run/Stream parity tests, diagnostics/OTel correlation, and contract gates through `RuntimeRecorder`.
- Add a new agent-mode example and update required documentation before implementing its runtime smoke path.
- Do not add a second state store, workspace filesystem, artifact content service, ACL, garbage collector, hosted control plane, or global queue.

## Capabilities

### New Capabilities

- `checkpoint-history-and-workspace-provenance`: Reference-only checkpoint lineage/history/branch/replay and workspace change-set/integrity provenance semantics.

### Modified Capabilities

- `agent-runtime-protocol-contract`: Add checkpoint lineage and workspace provenance fields and deterministic validation while preserving the six-object lifecycle and source ownership.
- `unified-state-and-session-snapshot-contract`: Extend protocol projection metadata for restore source, lineage, replay identity, and workspace integrity without changing manifest or restore semantics.
- `diagnostics-replay-tooling`: Add versioned checkpoint/workspace provenance fixtures and deterministic drift classifications while preserving mixed historical fixture compatibility.

## Impact

- `core/types`: additive protocol DTOs, enums, validators, and projection helpers.
- `orchestration/snapshot`: pure manifest-to-protocol provenance mapping; no storage interface changes.
- `orchestration/composer` and source adapters: pass restore/branch/replay/workspace context into projections without taking ownership.
- `observability/event` and `observability/trace`: nullable bounded provenance correlation through `RuntimeRecorder` only.
- `tool/diagnosticsreplay` and shell/PowerShell gates: new fixture parser, replay normalization, drift taxonomy, and parity checks.
- `examples/agent-modes`, README, runtime diagnostics, module boundaries, contract index, and roadmap documentation.
