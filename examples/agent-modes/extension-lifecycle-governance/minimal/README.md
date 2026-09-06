# extension-lifecycle-governance (minimal)

## Purpose

用离线、无副作用的 resolver/manifest 语义演示扩展治理最小闭环：资源来源优先级、digest provenance、非法 manifest 拒绝和同优先级冲突分类。

## Prerequisites

- Go 1.22+ and module dependencies resolved.
- Writable local `GOCACHE`; no external service or network is required.

## Real Runtime Path

- Semantic anchor: `extension.resource_resolution_manifest_admission`.
- Runtime path evidence: `extension,runtime/config,adapter/manifest,adapter/capability,observability/event`.
- Admission is manifest-first and reuses runtime readiness/policy; denied activation remains side-effect-free.

## Semantic Anchor

`extension.resource_resolution_manifest_admission`

## Runtime Path

`extension,skill/loader,adapter/manifest,adapter/capability,runtime/config,observability/event`

## Expected Output/Verification

- `verification.mainline_runtime_path=ok`
- `verification.semantic.anchor=extension.resource_resolution_manifest_admission`
- `verification.semantic.governance=baseline`

Expected markers:

- `extension_resource_discovered`
- `extension_precedence_resolved`
- `extension_manifest_validated`
- `extension_invalid_manifest_rejected`
- `extension_conflict_classified`

## Failure/Rollback Notes

本示例只读取本地 fixture，不加载扩展代码、不访问网络、不写入 Session/Artifact。若 resolver 行为变化，回退本目录与对应 replay fixture；运行时 feature gate 关闭时不改变现有 Skill/Adapter 路径。

## Verification

Replay fixture：`extension_lifecycle_governance.v1`。实现阶段需补充 `go run ./examples/agent-modes/extension-lifecycle-governance/minimal` 的 marker 输出和对应 gate。
