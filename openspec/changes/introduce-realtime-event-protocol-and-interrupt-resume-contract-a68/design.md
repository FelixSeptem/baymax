## Context

当前主干已经具备：
- A56：ReAct Run/Stream parity 与工具调用闭环；
- A58：跨域决策解释链；
- A67（进行中）：plan notebook + plan-change hook。

但 realtime 路径仍缺统一合同层：事件类型定义、序列推进、幂等去重、interrupt/resume 恢复游标及错误分层在不同入口下容易产生语义漂移。roadmap 已将 A68 定位为“实时协议专项，并带 `realtime_control_plane_absent` 边界断言”，因此本设计聚焦库内协议与执行接缝，不引入平台化控制面。

## Goals / Non-Goals

**Goals:**
- 定义 realtime event envelope 与 canonical event taxonomy。
- 定义 interrupt/resume 状态机、恢复游标与幂等去重合同。
- 定义配置治理：`runtime.realtime.protocol.*`、`runtime.realtime.interrupt_resume.*`。
- 定义可观测与回放：A68 additive 字段、`realtime_event_protocol.v1` fixture、drift taxonomy。
- 保证 equivalent workload 下 Run/Stream 语义等价。

**Non-Goals:**
- 不引入托管实时网关、连接路由服务、平台化会话控制面或 SaaS 运维面板。
- 不新增平行 ReAct loop 或替代 A56/A67 的主链路语义。
- 不在 A68 内推进性能专项（A64 负责）或示例收口（A62 负责）。

## Decisions

### Decision 1: 采用固定事件信封 + canonical 类型集

- 方案：统一事件信封字段（`event_id/session_id/run_id/seq/type/ts/payload`）与类型集（request/delta/interrupt/resume/ack/error/complete）。
- 备选：各入口自由定义事件结构后再做适配。
- 取舍：统一信封是 replay 可比对、门禁可阻断的前提。

### Decision 2: 序列语义采用“单调 seq + gap 检测 + 幂等去重”

- 方案：
  - 每会话 `seq` 单调递增；
  - 对重复 `event_id` 或 dedup key 做幂等吸收；
  - gap 检测触发规范化错误分类。
- 备选：仅 best-effort 事件流，不做强序列断言。
- 取舍：best-effort 会导致 interrupt/resume 回放不可验证。

### Decision 3: interrupt/resume 固定状态机，不新增第二套终止通道

- 方案：
  - interrupt 进入受控冻结状态并记录 resume cursor；
  - resume 仅从合法游标恢复，不可跨越未确认边界；
  - 终态分类继续复用 A56 taxonomy。
- 备选：resume 直接重启执行且忽略游标一致性。
- 取舍：游标一致性是等价恢复与幂等保证的核心。

### Decision 4: 错误分层复用现有 runtime 错误治理

- 方案：realtime 错误映射到既有 transport/protocol/semantic 分层，新增 A68 专属 reason code 仅做 additive 扩展。
- 备选：引入独立 realtime 错误域。
- 取舍：独立域会放大解释链分叉风险。

### Decision 5: fixture-first + 独立 gate + 边界断言

- 方案：新增 `realtime_event_protocol.v1` 与 drift taxonomy；新增 `check-realtime-protocol-contract.*` 并接入质量门禁；强制 `realtime_control_plane_absent`。
- 备选：仅依赖 integration 测试，不设独立 gate。
- 取舍：独立 gate 对同域漂移拦截更稳定，且与主线治理一致。

## Risks / Trade-offs

- [Risk] 中断恢复状态接线增加主循环复杂度。  
  -> Mitigation: 将 A68 变更限定在事件边界与状态边界，不改写 A56 终止路径。

- [Risk] 去重规则过严导致合法事件被误判重复。  
  -> Mitigation: dedup key 规范化 + 负向回归测试（近似重复但语义不同）。

- [Risk] Run/Stream 并发时序导致 parity 假阳性失败。  
  -> Mitigation: parity 断言聚焦语义等价而非字节级序列完全一致。

- [Risk] A68 与 A67/A67-CTX 产生边界重叠。  
  -> Mitigation: A68 只定义实时协议与 interrupt/resume，context 组织仍归 A67-CTX。

## Migration Plan

1. 配置层：在 `runtime/config` 增加 `runtime.realtime.protocol.*` 与 `runtime.realtime.interrupt_resume.*`，实现 fail-fast 与热更新回滚。
2. 协议层：定义事件信封、canonical 类型、序列与 dedup 规则。
3. 执行层：在 `core/runner` 接入 interrupt/resume 状态机与恢复游标。
4. 观测层：在 diagnostics/recorder 增加 A68 additive 字段与错误码映射。
5. 回放层：新增 `realtime_event_protocol.v1` fixture、drift 分类、mixed fixture 兼容测试。
6. 门禁层：新增 `check-realtime-protocol-contract.sh/.ps1`，接入 `check-quality-gate.*` 并实现 `realtime_control_plane_absent` 断言。
7. 文档层：同步 runtime config/diagnostics、contract index、roadmap 与 README。

回滚策略：
- 配置回滚：热更新非法配置自动回滚到上一个有效快照；
- 功能回滚：关闭 `runtime.realtime.protocol.enabled` 与 `runtime.realtime.interrupt_resume.enabled` 恢复现有路径；
- 数据兼容：新增字段保持 additive，旧消费者可安全忽略。

## Open Questions

- None. A68 按 roadmap 一次性收口 realtime 协议与 interrupt/resume 同域需求，不再拆平行提案。
