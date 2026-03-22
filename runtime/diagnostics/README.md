# runtime/diagnostics 组件说明

## 功能域

`runtime/diagnostics` 提供统一诊断数据模型与查询能力：

- 调用记录：`CallRecord`
- 运行摘要：`RunRecord`
- mailbox 记录：`MailboxRecord`
- 技能记录：`SkillRecord`
- 热更新记录：`ReloadRecord`

当前稳定查询接口：
- `RecentCalls`
- `RecentRuns`
- `RecentReloads`
- `RecentMailbox`
- `QueryMailbox`
- `MailboxAggregates`
- `QueryRuns`
- `RecentSkills`
- `TimelineTrends`
- `CA2ExternalTrends`

## 架构设计

`Store` 使用有界内存结构维护多类记录，并支持：

- 容量动态调整（Resize）
- 幂等去重（run/skill idempotency key）
- 统一 run 查询（`QueryRuns`：多维过滤 + 分页 + 排序 + opaque cursor）
- timeline 聚合（phase 统计、P95 延迟）
- 趋势查询（`TimelineTrends`、`CA2ExternalTrends`）
- 多代理 additive 摘要字段（含 `async_await_*`、`async_reconcile_*`、`collab_*`、`scheduler_*`、`mailbox_*`）

该包只负责数据模型和聚合算法，不负责事件订阅。

## 关键入口

- `store.go`

## 边界与依赖

- 诊断写入统一由 `observability/event.RuntimeRecorder` 驱动。
- 业务模块不应直接构造跨域写路径绕开 RuntimeRecorder。
- 新增诊断字段时，需要同步代码、文档与契约测试索引。

## 配置与默认值

- Store 容量与趋势窗口默认值来自 `runtime/config` diagnostics 子域。
- 未显式配置分页时 `QueryRuns` 默认 `page_size=50`，上限 `200`。
- 未显式配置分页时 `QueryMailbox` 默认 `page_size=50`，上限 `200`。
- 排序默认 `time desc`，游标为 opaque token。

## 可观测性与验证

- 关键验证：`go test ./runtime/diagnostics -count=1`。
- 回归重点：幂等写入、重复事件收敛、趋势聚合确定性。
- 与 integration 契约共同验证 run/team/workflow/task 查询语义。

## 扩展点与常见误用

- 扩展点：新增聚合维度、趋势查询视图、导出格式适配。
- 常见误用：业务模块绕过 RuntimeRecorder 直接写 Store。
- 常见误用：调整 cursor 语义但未同步兼容测试，导致分页漂移。
