## Why

Context Assembler 已完成 CA1/CA2，但在高负载与长上下文场景下缺少系统化的内存压力控制与恢复机制，导致上下文膨胀、回放恢复和降级路径仍依赖临时策略。现在进入 CA3 可以在不改变 runner 主状态机的前提下，建立可配置、可观测、可恢复的压力治理基线，为后续 CA4/HITL/A2A 提供稳定底座。

## What Changes

- 实现 CA3 内存压力控制与恢复能力：分级压力响应、batch squash/prune、spill/swap、本地文件回填。
- 压力阈值采用双触发机制：百分比阈值 + 绝对 token 阈值（任一满足即触发），并支持按阶段配置。
- 引入 `critical` 与 `immutable` 保护标记，确保关键内容不被压缩/删除。
- 紧急区默认拒绝低优先级新加载请求，高优先级请求允许降级通过。
- 保障单进程内 `cancel/retry/replay` 恢复一致性。
- 增加 CA3 诊断字段（停留时长、触发次数、压缩率、溢出/回填计数），并继续复用现有 `runtime/diagnostics`。
- 强化契约测试，要求 Run/Stream 在压力分级决策结果上语义一致。
- 文档与实现保持一致：README、runtime-config-diagnostics、context-assembler-phased-plan、development-roadmap 同步更新。

## Capabilities

### New Capabilities
- `context-assembler-memory-pressure-control`: 定义 CA3 分级压力治理、squash/prune、spill/swap 与恢复一致性语义。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 扩展 CA3 配置项与 run 诊断字段（不新增独立 API）。
- `context-assembler-stage-routing`: 增加 CA3 下 Run/Stream 压力分级决策语义一致性约束。

## Impact

- 影响模块：`context/assembler`、`context/journal`、`runtime/config`、`runtime/diagnostics`、`observability/event`。
- 不新增外部服务依赖；`spill/swap` 本期仅文件后端实现，DB/对象存储接口预留不实现。
- API 影响：仅扩展 `RecentRuns` 返回字段，保持向后兼容。
- 质量门禁：`go test ./...`、`go test -race ./...`、`golangci-lint`、docs consistency 必须通过。
