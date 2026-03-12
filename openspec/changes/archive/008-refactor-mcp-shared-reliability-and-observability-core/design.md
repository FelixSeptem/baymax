## Context

当前 `mcp/http` 与 `mcp/stdio` 在调用链中分别维护重试/backoff、超时包装、事件发射、诊断写入映射等逻辑，导致同一可靠性策略在两个 transport 上演进时容易出现语义偏差。近期已完成运行时配置、诊断 single-writer 与并发质量基线，下一步应优先收敛 MCP 传输层内部重复逻辑，降低维护与回归成本。

约束条件：
- 共享核心必须强封装，仅供 `mcp` 包内部复用，不作为对外 API。
- 不扩展到 example 批次与多 provider 实现（后续里程碑单独提案）。
- 需要提供可复现的重复代码下降度量作为验收依据。

## Goals / Non-Goals

**Goals:**
- 在 `mcp/internal/*` 建立共享可靠性与可观测性核心，统一 retry/timeout/event/diag 语义。
- 让 `mcp/http` 与 `mcp/stdio` 仅保留 transport-specific 逻辑。
- 增加跨 transport 契约测试，保证错误分类、诊断字段与关键事件语义一致。
- 输出重复逻辑下降比例（相对百分比）并纳入验收。

**Non-Goals:**
- 不改变 `mcp/http` 与 `mcp/stdio` 的对外接口形状。
- 不引入新的 transport 协议。
- 不包含 examples 扩容与多 provider 功能实现。

## Decisions

### 1) 共享核心路径采用 `mcp/internal/*` 强封装
- 决策：共享执行逻辑放在 `mcp/internal/reliability`（或等价命名）与 `mcp/internal/observability`，禁止非 `mcp/*` 目录引用。
- 理由：满足“仅包内部复用”的要求，避免过早抽象成全局 runtime 公共库。
- 备选：放在 `runtime/*`。
  - 放弃原因：会扩大职责边界并增加跨域耦合风险。

### 2) 抽象统一调用执行骨架（invoke loop）
- 决策：共享核心提供“重试+超时+backoff+事件模板+诊断回写”的执行骨架；transport 仅提供具体调用函数与连接管理差异。
- 理由：最大化去重收益且保持传输层可读性。
- 备选：只抽事件与诊断，保留各自重试循环。
  - 放弃原因：仍保留大量重复和漂移风险。

### 3) 重复逻辑下降比例采用静态统计脚本输出
- 决策：引入可重复执行的统计脚本，基于重构前基线（记录在 docs/change artifacts）与重构后数据计算相对下降百分比。
- 理由：避免主观估计，支持 PR/归档复核。
- 备选：人工 review 估算。
  - 放弃原因：不可重复、不可审计。

### 4) 契约测试以 transport 对照表驱动
- 决策：定义统一场景矩阵（retryable/non-retryable/timeout/backpressure/reconnect），对 `http` 与 `stdio` 并行执行并比较标准化输出。
- 理由：直接覆盖“语义一致性”而非实现细节。
- 备选：分散在各自包单测。
  - 放弃原因：难以保证跨 transport 一致性。

## Risks / Trade-offs

- [Risk] 内部抽象过度导致 transport 特性表达受限 → Mitigation: 明确“共享骨架 + transport hook”边界，保留扩展钩子。
- [Risk] 去重比例达标但可读性下降 → Mitigation: 以契约测试和代码审查同时约束，拒绝“为去重而去重”。
- [Risk] 重构引入回归（重试停止条件/事件顺序） → Mitigation: 增加跨 transport 契约测试与回归金丝雀场景。

## Migration Plan

1. 建立 `mcp/internal/*` 共享核心并补充单测。
2. 改造 `mcp/http` 接入共享核心，保持行为不变。
3. 改造 `mcp/stdio` 接入共享核心，保持行为不变。
4. 增加跨 transport 契约测试与重复逻辑比例报告。
5. 更新 README/docs 边界说明并执行全量验证。

## Open Questions

- 重复逻辑比例统计脚本最终使用 `cloc`、`dupl` 还是仓库内自定义统计器（实现阶段根据可维护性定案）。