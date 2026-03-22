## Context

当前 mailbox 能力存在三层不一致：

1. 契约层已定义 mailbox 为多代理 sync/async/delayed 主链路（A30 及后续）。
2. 配置层已提供 `mailbox.*` 完整字段（backend/path/retry/ttl/query）与 fail-fast 校验。
3. 实现层在关键路径仍按调用创建临时 in-memory bridge，未形成 runtime 级共享 mailbox 生命周期，也缺少主链路诊断写入闭环。

这导致：
- `mailbox.*` 的不少字段只在配置对象中存在，未直接影响主链路行为；
- `QueryMailbox`/`MailboxAggregates` 在主链路上的可观测性不足；
- memory/file parity 契约与运行时默认行为之间存在“语义可测、路径未接线”的落差。

## Goals / Non-Goals

**Goals:**
- 在 runtime 主链路启用共享 mailbox wiring，避免 per-call 临时 bridge。
- 将 `mailbox.*` 配置转化为可运行、可回滚、可观测的有效语义。
- 为 mailbox 主链路补齐 diagnostics 写入与 query/aggregate 可追踪性。
- 通过 shared gate 阻断 mailbox wiring 回退与语义漂移。

**Non-Goals:**
- 不引入外部 MQ 或平台化控制面。
- 不改变 A32 async-await 终态收敛主契约。
- 不替代 A34 的 canonical invoke API 收口目标（A35 假设 A34 已确定入口面）。

## Decisions

### 1) 共享 mailbox 生命周期由 Composer 管理路径托管
- 决策：在 Composer managed 路径引入 mailbox runtime 实例及签名刷新机制；collab/scheduler 通过注入使用共享 bridge。
- 原因：Composer 已托管 scheduler/recovery 生命周期与热更新刷新，复用同一治理模型成本最低。
- 备选：继续在 collab/scheduler 内部按调用创建 bridge。拒绝原因：配置难生效，状态无法共享，观测弱。

### 2) `mailbox.enabled=false` 仍使用共享 memory mailbox
- 决策：关闭开关不表示“禁用 mailbox 契约”，而是使用共享 memory backend 作为默认运行形态。
- 原因：A30 后 mailbox 已为 canonical 协调路径，关闭开关更适合作为“非持久化后端选择”，不是“绕过 mailbox”。
- 备选：`enabled=false` 时退回 direct path。拒绝原因：与 A34 canonical-only 方向冲突。

### 3) `backend=file` 初始化失败执行可观测回退
- 决策：`mailbox.enabled=true && backend=file` 初始化失败时回退到 memory，并记录 fallback reason（不 silent）。
- 原因：保持 runtime 可用性并避免启动硬失败，同时通过诊断提供可追踪治理信号。
- 备选：file 初始化失败直接 fail-fast。拒绝原因：对本地/开发场景恢复性差，且与现有 scheduler fallback 策略不一致。

### 4) 诊断首批覆盖 publish 主路径
- 决策：首批在 command/result publish 与 delayed command publish 路径落 `RecordMailbox`，确保 query/aggregate 看到真实业务流量。
- 原因：覆盖主价值链路且实现复杂度可控；ack/nack/requeue 深度观测可后续增量提案补齐。
- 备选：一次性覆盖全部 mailbox lifecycle。拒绝原因：范围过大、与当前提案“接线收口”目标不匹配。

### 5) 质量门禁加入 wiring 契约阻断
- 决策：shared multi-agent gate 增加 mailbox runtime wiring 套件，验证：
  - 配置接线生效（env/file/default）；
  - file->memory fallback 语义；
  - Run/Stream 等价与 memory/file parity；
  - 诊断 query/aggregate 可追踪。
- 原因：只有 gate 阻断才能防止后续回退到 per-call 临时 bridge。
- 备选：仅依赖代码审查与文档。拒绝原因：不可回归验证。

## Risks / Trade-offs

- [Risk] 共享 mailbox 状态增加跨调用耦合，可能影响现有测试假设  
  -> Mitigation: 显式区分 managed/shared 与 test-local bridge；对受影响用例更新 fixture 与清理边界。

- [Risk] fallback 行为掩盖 file backend 初始化问题  
  -> Mitigation: 必须记录 deterministic fallback reason，并在 diagnostics 中可查询。

- [Risk] 新增 wiring 刷新逻辑与 runtime hot reload 交互复杂  
  -> Mitigation: 采用签名比较 + 原子替换模式，失败保持 last-known-good 快照。

## Migration Plan

1. 引入 mailbox runtime holder（实例 + backend/fallback 元信息 + 配置签名）。
2. 将 collab/scheduler 调用路径切换为可注入 bridge/provider，managed 模式注入共享实例。
3. 接入 `runtime/config` mailbox 配置解析结果与刷新流程（启动 + reload）。
4. 在 publish 主路径接入 `RecordMailbox` 写入，并补 query/aggregate 断言测试。
5. 增加 shared gate/quality gate wiring suites 与脚本映射。
6. 同步更新 README/roadmap/runtime-config-diagnostics/mainline-index 文档。

回滚策略：
- 保留测试与非 managed 场景可显式传入本地 bridge；
- 若共享 wiring 引发回归，可在变更窗口内回退为 managed 路径外的现有行为并保留配置字段。

## Open Questions

无阻塞项，按推荐值执行：
- `mailbox.enabled=false` -> 共享 memory mailbox
- `backend=file` 初始化失败 -> fallback 到 memory 并记录 reason
- 诊断首批覆盖 publish 主路径
