## Why

在 A63（命名/文档收敛）与 A64（性能/工程优化）推进后，主干 contract 能力已较完整，但交付侧仍缺少统一、可回归、可迁移的 agent 模式示例矩阵。当前示例分散且覆盖深度不一致，导致团队从 PoC 到生产落地的迁移成本偏高，且难以稳定复用主线 gate 口径。

## What Changes

- 新增 A62 主合同：建设 `agent-mode example pack`，将主要 agent 模式沉淀为统一目录、统一矩阵、统一门禁、统一迁移手册。
- 固化模式覆盖范围（PocketFlow + Baymax 扩展）：
  - PocketFlow：`agent/workflow/rag/mapreduce/structured-output/multi-agents`。
  - Baymax：`skill/mcp/react/hitl/context/sandbox/realtime`。
- 固化示例形态：每个模式必须提供 `minimal + production-ish` 双档示例。
- 固化主干流程覆盖：补齐 mailbox `sync/async/delayed/reconcile`、task-board `query/control`、scheduler `qos/backoff/dead-letter`、readiness/admission 降级链路示例。
- 固化自定义 adapter 覆盖：补齐 `mcp/model/tool/memory` 四类接入及 `manifest/capability/profile-replay/health-readiness-circuit` 治理链路示例。
- 新增 `example -> production` 迁移手册（`PLAYBOOK.md`）与每个 `production-ish` README 的 `prod delta` 检查清单。
- 清理历史示例遗留占位：`examples/` 下既有示例中的 `TODO/TBD/FIXME/待补` 占位必须清零，并迁移为可追踪矩阵项或任务项。
- 新增并接入示例专项门禁：
  - `check-agent-mode-examples-smoke.sh/.ps1`
  - `check-agent-mode-pattern-coverage.sh/.ps1`
  - `check-agent-mode-migration-playbook-consistency.sh/.ps1`
  - `check-agent-mode-legacy-todo-cleanup.sh/.ps1`
- 新增“提案级示例影响评估”约束：
  - 后续涉及行为/配置/契约变化的提案，必须显式判断是否需要修改现有 example 或新增 example；
  - 评估结果必须在提案工件中落地为三选一：`新增示例`、`修改示例`、`无需示例变更（附理由）`。
- 新增 a62 与 a69 的依赖边界：
  - a62 非 context 主题任务（如 workflow/rag/mcp/hitl/sandbox/realtime）可并行推进；
  - `context-governed-reference-first` 及其验收必须以后置 A69（context compression production hardening）收敛为准，并引用 A69 的 gate/replay 输出。
- 保持硬约束：示例层仅复用既有 contract 输出，禁止定义平行语义；不引入平台化控制面；Run/Stream 语义保持等价。

## Capabilities

### New Capabilities
- `delivery-usability-agent-mode-example-pack-contract`: 统一定义模式矩阵、示例双档、主干流程覆盖、自定义 adapter 覆盖、迁移手册与阻断门禁。

### Modified Capabilities
- `go-quality-gate`: 增加 A62 示例 smoke、模式覆盖矩阵校验、迁移手册一致性校验的阻断接线与 shell/PowerShell parity 要求。

## Impact

- 代码与示例目录：`examples/agent-modes/*`（新增/改造）。
- 质量门禁脚本：`scripts/check-agent-mode-examples-smoke.*`、`scripts/check-agent-mode-pattern-coverage.*`、`scripts/check-agent-mode-migration-playbook-consistency.*`、`scripts/check-agent-mode-legacy-todo-cleanup.*`、`scripts/check-quality-gate.*`。
- 测试与回放：示例相关 integration smoke、diagnostics/replay 夹具与 drift 分类断言。
- 文档与索引：`README.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`examples/agent-modes/MATRIX.md`、`examples/agent-modes/PLAYBOOK.md`。
