## Why

The roadmap places external Extension authoring conformance immediately after the completed provider and durable-operation baselines. Baymax already has extension lifecycle, resource resolution, manifest/capability admission, policy, sandbox, allowlist, and replay primitives, but an extension author still lacks one deterministic workflow that explains why a package is rejected, degraded, or isolated. Without that feedback loop, independently authored extensions can drift from the existing contracts while local tests remain green.

This change is therefore the next roadmap item: it turns the existing contracts into an offline, fixture-driven authoring conformance surface without introducing a package manager, hosted registry, or second extension runtime.

## What Changes

- Add a versioned, transport-neutral authoring conformance case format covering descriptor/manifest metadata, source precedence, digest provenance, capability negotiation, admission, lifecycle ordering, and generation reload behavior.
- Add an offline conformance harness that evaluates cases against the existing extension governance owners and returns deterministic pass/fail records with bounded phase, field, reason, and remediation metadata.
- Cover positive, negative, boundary, and recovery workflows: valid activation; missing or malformed fields; compatibility and capability mismatch; optional downgrade; equal-precedence digest conflict; timeout, panic, invalid result, and finalize failure; stale-generation suppression; and failed-reload rollback.
- Prove that authoring conformance cannot bypass readiness, policy, sandbox, allowlist, egress, `RuntimeRecorder`, or authoritative Run/Stream terminal semantics.
- Add replay fixtures and shell/PowerShell parity gates for deterministic ordering, failure taxonomy, remediation output, idempotency, and Run/Stream equivalence.
- Keep diagnostics additive, nullable, bounded, and free of source code, credentials, reasoning, workspace content, or unbounded payloads.
- Update the contract index, module-boundary and diagnostics documentation, roadmap status, and authoring guidance with the new harness and rollback path.
- **Example Impact Assessment:** `无需示例变更（附理由）`。The change adds offline conformance fixtures and author feedback only; it does not alter `examples/agent-modes` runtime behavior or introduce a new mode. Any future example change must first update `MATRIX.md` and the corresponding mode README.

## Capabilities

### New Capabilities

- `external-extension-authoring-conformance`: deterministic, offline conformance cases, author-facing failure feedback, replay compatibility, and policy/sandbox boundary proofs for externally authored extensions.

### Modified Capabilities

无。现有 `extension-lifecycle-governance`、`deterministic-resource-resolution`、adapter capability/manifest、policy、sandbox、allowlist 与 diagnostics 合同作为被测 owner；本变更不修改其既有 requirement，只增加针对外部 authoring workflow 的可验证 conformance capability。

## Impact

- Affected code: `extension`, `adapter/manifest`, `adapter/capability`, policy/readiness and sandbox/allowlist integration seams, `observability/event`, `tool/diagnosticsreplay`, and new integration conformance test helpers.
- Affected artifacts: versioned fixture files, replay normalizers, shell/PowerShell gate scripts, and contract-test index entries.
- No new runtime configuration keys, persistence migrations, provider dependencies, network access, package manager, registry, or hosted service.
- Runtime behavior changes are not expected unless a conformance fixture demonstrates an existing contract drift; any such fix must be minimal, additive, and preserve Run/Stream parity.
- Verification commands: focused extension and integration tests; replay and dedicated gates on both shells; `go test ./...`; `go test -race ./...`; `golangci-lint run --config .golangci.yml`; `pwsh -File scripts/check-quality-gate.ps1`; `pwsh -File scripts/check-docs-consistency.ps1`; and `openspec validate --all`.

## Example Impact Assessment

无需示例变更（附理由）

理由：The change adds offline conformance fixtures and author feedback only; it does not alter `examples/agent-modes` runtime behavior or introduce a new mode. Any future example change must first update `MATRIX.md` and the corresponding mode README.
