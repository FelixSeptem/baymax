## Context

当前多代理链路在契约层已经收敛为 mailbox 统一协调（A30+），但 `orchestration/invoke` 仍同时保留两类入口：

- `MailboxBridge.InvokeSync/InvokeAsync`（主线推荐）
- `InvokeSync/InvokeAsync`（deprecated，但仍被 bridge 内部调用）

这会导致“公共面与实现面不一致”：
- 文档和契约声明 mailbox 是 canonical；
- 代码实现仍依赖 direct 导出 API；
- 质量门禁未显式阻断 legacy API 回流。

在 A33（bounded retry）进行中的前提下，A34 需要独立收敛调用入口，不改变 A33/A32 已定义语义。

## Goals / Non-Goals

**Goals:**
- 固定 mailbox bridge 为 sync/async/delayed 唯一 canonical 调用入口。
- 退场 legacy direct invoke 公共 API，消除“deprecated but in-use”中间态。
- 保持 Run/Stream、memory/file、replay idempotency 契约等价。
- 增加 canonical-only 质量门禁，阻断 direct invoke 回流。

**Non-Goals:**
- 不引入外部 MQ 或平台化控制面能力。
- 不修改 A32 async-await callback/reconcile/timeout 收敛语义。
- 不扩展新的多代理执行模式（仅做入口收口与治理）。

## Decisions

### 1) API 收口策略：硬切 legacy direct invoke
- 决策：移除/退场 `invoke.InvokeSync` 与 `invoke.InvokeAsync` 作为对外 canonical 公共入口，不再提供兼容 wrapper。
- 原因：仓库尚未对外稳定发布，且已明确无需为 A11/A12/A13 历史调用面保兼容，硬切能最快消除中间态。
- 备选：保留 deprecated wrapper。拒绝原因：会延长双入口共存窗口，增加回归与认知成本。

### 2) Bridge 内部实现改为私有路径
- 决策：`MailboxBridge` 内部不再调用已退场导出函数；改用 `orchestration/invoke` 私有 helper（仅包内可见）实现 submit/wait 与 async submit。
- 原因：保证 bridge 真正成为唯一稳定调用面，同时保留必要复用，避免 duplicated logic。
- 备选：把所有逻辑内联到 bridge。拒绝原因：可读性与可测试性下降。

### 3) 契约升级为 canonical-only
- 决策：
  - sync 契约要求公共调用面仅 mailbox path；
  - async 契约要求 legacy direct report-sink 退出公共契约面；
  - mailbox 契约补充统一入口强约束。
- 原因：与现有 roadmap 和 mainline index 口径一致，避免“规范允许双入口”。
- 备选：维持 deprecated 描述。拒绝原因：无法彻底关闭中间态。

### 4) 质量门禁加入防回流检查
- 决策：在 shared contract gate + quality gate 增加 canonical-only 检查，至少覆盖：
  - legacy direct invoke API 暴露回归；
  - 跨模块重新调用 legacy direct 路径；
  - mailbox 主线路径 Run/Stream 等价与 replay 稳定。
- 原因：单靠文档约束无法防回归，必须 gate 化。
- 备选：仅依赖 code review。拒绝原因：不可重复验证。

## Risks / Trade-offs

- [Risk] 对内部依赖 direct API 的测试或示例产生编译影响  
  -> Mitigation: 在同一提案内完成调用点迁移与测试重构，并以全仓 `go test ./...` 验证。

- [Risk] 收口过程中引入 sync/async 行为漂移  
  -> Mitigation: 复用并增强 A11/A12/A30/A31/A32 既有契约测试矩阵，确保 Run/Stream 与 replay 语义不变。

- [Risk] 质量门禁规则过严导致误报  
  -> Mitigation: 规则仅针对特定 legacy symbol 与公开导出面，先以 fixture 覆盖再接入阻断。

## Migration Plan

1. 修改 `orchestration/invoke` API 面：退场 legacy direct public invoke 入口，保留 bridge 入口。
2. 重构 bridge 内部执行逻辑到私有 helper，保持错误分类与 retryable 判定稳定。
3. 迁移 `orchestration/collab`、`orchestration/scheduler` 等调用点与相关测试。
4. 更新 integration/shared gate 与 quality gate，加入 canonical-only 回归阻断。
5. 同步文档（README/roadmap/mainline-index/module README）删除中间态描述。
6. 执行验证：`go test ./...`、`go test -race ./...`、`golangci-lint`、docs consistency、shared contract gate。

回滚策略：
- 若发布前发现兼容性问题，可在未归档前回退 A34 变更；归档后不再恢复 legacy direct 公共入口。

## Open Questions

无阻塞项。按既定推荐值执行：
- 采用硬切（不保留 wrapper）
- 纳入 canonical-only gate 阻断
- A33 合并后独立实施 A34，避免同目录冲突
