# context 组件说明

## 功能域

`context` 提供上下文装配链路（语义阶段）所需能力：

- 装配编排：`context/assembler`
- journal 存储：`context/journal`
- guard 校验：`context/guard`
- stage2 provider 适配：`context/provider`

Canonical 架构入口：`docs/runtime-harness-architecture.md`

## 架构设计

当前实现由 `Assembler` 统一编排多阶段流程：

- context-prefix-and-journal-baseline：prefix hash + guard + journal intent/commit
- context-stage2-routing-and-disclosure：stage2 规则/agentic 路由 + provider 拉取 + tail recap
- context-pressure-compaction-and-swapback：压力分区、压缩、prune、spill、semantic compaction
- context-production-hardening-and-threshold-governance：阈值覆盖与触发来源追踪（通过压力阶段统计字段体现）

`provider` 子域负责 file/http/rag/db/elasticsearch 的检索适配和错误分层（`transport|protocol|semantic`）。

## 关键入口

- `assembler/assembler.go`
- `assembler/context_pressure_recovery.go`
- `journal/storage.go`
- `guard/guard.go`
- `provider/provider.go`

## 边界与依赖

- `context/*` 不直接依赖 provider 官方 SDK；模型能力应通过 `model/*` 间接复用。
- stage2 错误层、reason code、hint 元数据需保持契约稳定，供 diagnostics 聚合。
- 该域只生成标准结果与事件，不直接写 `runtime/diagnostics` 存储。

## 配置与默认值

- 语义阶段阈值与策略默认值由 `runtime/config` 提供，`context/*` 只消费快照。
- Stage2 外部检索默认采用 best-effort，可按治理策略切换 fail-fast。
- 压力压缩与阈值治理默认走保守参数，避免过度剪裁。

## 可观测性与验证

- 关键验证：`go test ./context/assembler ./context/guard ./context/journal ./context/provider -count=1`。
- 可观测指标包括 phase 延迟、pressure level、stage2 provider 错误分层。
- 回归重点是 run/stream 语义一致与 reason taxonomy 稳定。

## 扩展点与常见误用

- 扩展点：新增 stage2 provider adapter、扩展压力压缩 reranker/scorer、接入新阈值治理策略。
- 常见误用：在 assembler 内直接做 provider SDK 调用，绕过 provider 抽象层。
- 常见误用：无诊断标注地引入压缩策略变更，导致线上排障困难。
