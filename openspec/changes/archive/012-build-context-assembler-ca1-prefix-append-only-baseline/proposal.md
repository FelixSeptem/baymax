## Why

Context Assembler 涉及 prefix 一致性、append-only、规则防护、可中断恢复与观测闭环，若一次性实现全量能力会显著放大变更域与回归风险。先交付 CA1 基线可在不破坏现有 runner/tool/stream 语义前提下建立稳定基础，并为后续 CA2/CA3/CA4 分期演进提供可验证约束。

## What Changes

- 新增 `context/assembler` CA1 基线：作为 runner 的 pre-model 阶段钩子，构建 immutable prefix 与最小上下文拼装流程。
- 新增 `context/journal` 本地文件 append-only 事件日志（JSONL），仅追加不重排，并保留 intent/commit 记录。
- 新增 `context/guard` 基础防护：prefix hash 校验、基础 schema 校验、敏感字段脱敏；默认 fail-fast。
- 扩展 `runtime/config` 最小配置：`context_assembler.enabled`（默认启用）、`journal_path`、`prefix_version`、`guard.fail_fast`。
- 扩展 `runtime/diagnostics` 最小字段：`prefix_hash`、`assemble_latency_ms`、`assemble_status`、`guard_violation`。
- 预留存储后端扩展接口（db 占位），CA1 仅实现本地文件方案，不接入数据库。
- 同步更新关联文档（README、roadmap、runtime-config-diagnostics、v1-acceptance、context-assembler-phased-plan）。

## Capabilities

### New Capabilities
- `context-assembler-baseline`: 定义 pre-model context assembler 的 CA1 基线契约（immutable prefix、append-only journal、guard fail-fast、最小诊断字段）。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 扩展 context assembler 的最小配置项与诊断可观测契约。

## Impact

- 受影响代码：`core/runner`、`core/types`、`runtime/config`、`runtime/diagnostics`、`observability/event`。
- 新增模块：`context/assembler`、`context/journal`、`context/guard`。
- 测试影响：新增 assembler 契约与回归测试（run/stream 语义兼容、append-only 与 prefix hash 稳定性、并发安全）。
- 文档影响：`README.md`、`docs/development-roadmap.md`、`docs/runtime-config-diagnostics.md`、`docs/v1-acceptance.md`、`docs/context-assembler-phased-plan.md`。
