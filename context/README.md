# context 组件说明

## 功能域

`context` 提供上下文装配链路（CA1-CA4）所需能力：

- 装配编排：`context/assembler`
- journal 存储：`context/journal`
- guard 校验：`context/guard`
- stage2 provider 适配：`context/provider`

## 架构设计

当前实现由 `Assembler` 统一编排多阶段流程：

- CA1：prefix hash + guard + journal intent/commit
- CA2：stage2 规则/agentic 路由 + provider 拉取 + tail recap
- CA3：压力分区、压缩、prune、spill、semantic compaction
- CA4：阈值覆盖与触发来源追踪（通过 CA3 统计字段体现）

`provider` 子域负责 file/http/rag/db/elasticsearch 的检索适配和错误分层（`transport|protocol|semantic`）。

## 关键入口

- `assembler/assembler.go`
- `assembler/ca3.go`
- `journal/storage.go`
- `guard/guard.go`
- `provider/provider.go`

## 边界与依赖

- `context/*` 不直接依赖 provider 官方 SDK；模型能力应通过 `model/*` 间接复用。
- stage2 错误层、reason code、hint 元数据需保持契约稳定，供 diagnostics 聚合。
- 该域只生成标准结果与事件，不直接写 `runtime/diagnostics` 存储。
