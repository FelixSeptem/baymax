## Context

S2 已在运行时实现工具权限/限流与模型 I/O 过滤阻断，并输出基础诊断字段（`policy_kind`、`namespace_tool`、`filter_stage`、`decision`、`reason_code`）。

当前缺口在于“可运营性”而非“可阻断性”：
- 缺少统一安全事件分类（taxonomy），跨模块 reason code 与严重级别口径未完全收敛；
- 缺少标准化告警触发契约，生产系统无法稳定订阅阻断事件；
- 缺少 S3 级独立门禁，无法在 CI 中单独阻断安全事件语义回归。

约束（已确认）：
- 告警仅覆盖 `deny`；
- 首期仅支持 callback sink；
- 严重级别采用 `low|medium|high` 三档；
- Run/Stream 安全事件语义必须强等价；
- 独立 CI job 作为 required-check 候选。

## Goals / Non-Goals

**Goals:**
- 定义统一 S3 安全事件 taxonomy（字段、枚举、reason code 归一化规则）。
- 建立 deny-only 告警触发契约与 callback 扩展接口。
- 在 runtime config 中引入 S3 事件与告警配置，保持 `env > file > default` 与热更新回滚语义。
- 将 S3 事件写入 diagnostics（增量字段、向后兼容）。
- 保证 Run/Stream 在等价输入下的安全事件与告警语义等价。
- 新增 `security-event-gate` 独立 CI 门禁。

**Non-Goals:**
- 不接入外部告警平台 SDK（如 PagerDuty/Slack/邮件网关）。
- 不在本阶段触发 `match` 告警。
- 不引入分布式事件总线或跨进程可靠投递协议。
- 不变更现有 S2 的 deny 决策策略。

## Decisions

### Decision 1: 采用统一 Security Event Envelope
- Choice: 新增统一事件结构（逻辑上）包含 `event_id/run_id/iteration/policy_kind/namespace_tool/filter_stage/decision/reason_code/severity/timestamp`。
- Rationale: 便于跨 tool governance 与 io filtering 共用观测与告警面。
- Alternative considered: 沿用各模块各自 payload。
- Rejected because: 无法稳定做统一聚合和门禁校验。

### Decision 2: 告警触发规则固定 deny-only
- Choice: 仅 `decision=deny` 触发告警 callback；`match|allow` 仅记录事件不告警。
- Rationale: 降低噪声与误报，先确保阻断事件的可运营闭环。
- Alternative considered: deny+match 全量告警。
- Rejected because: 首期信噪比差，增加运营负担。

### Decision 3: 首期告警 sink 仅支持 callback
- Choice: 在 runner/runtime 暴露 callback 接口，由 host 注入处理逻辑。
- Rationale: 保持 library-first，不绑定具体外部系统。
- Alternative considered: 内置 webhook/http sink。
- Rejected because: 引入额外依赖与失败语义复杂度。

### Decision 4: 严重级别三档映射
- Choice: severity 固定 `low|medium|high`；默认 deny 事件映射 `high`，并允许按 `policy_kind/reason_code` 扩展映射表。
- Rationale: 统一运营口径并保留后续扩展空间。
- Alternative considered: 五级或自由文本。
- Rejected because: 初期过度复杂或不可治理。

### Decision 5: Run/Stream 强等价契约
- Choice: 对等输入与配置下，Run/Stream 的 `policy_kind/decision/reason_code/severity` 语义必须等价。
- Rationale: 避免同策略在不同执行路径下行为分裂。
- Alternative considered: 仅约束 deny 结果。
- Rejected because: 会导致诊断与告警聚合不可比。

### Decision 6: 独立 CI 安全事件门禁
- Choice: 新增 `security-event-gate`，运行 S3 事件契约测试并作为 required-check 候选。
- Rationale: 安全事件语义回归需要单独可见和单独阻断。
- Alternative considered: 合并到现有 `test-and-lint`。
- Rejected because: 结果可见性不足，不利于 branch protection 治理。

## Risks / Trade-offs

- [Risk] callback 执行失败影响主流程稳定性。
  - Mitigation: callback 失败默认不改变既有 deny 决策，仅记录告警投递失败事件与 reason code。

- [Risk] deny-only 可能遗漏高风险 match 线索。
  - Mitigation: 保留 match 事件记录和趋势字段，为后续策略升级提供数据。

- [Risk] 事件字段扩展导致消费者解析差异。
  - Mitigation: 所有 S3 字段采用 additive 策略，不改变既有字段含义。

- [Risk] Run/Stream 等价测试覆盖不足导致隐性回归。
  - Mitigation: 在独立 gate 中强制执行契约测试，覆盖 permission/rate-limit/io-filter 三类来源。

## Migration Plan

1. 在 `runtime/config` 增加 S3 事件与告警配置字段，并接入 validate/default/build/hot-reload。
2. 在 `core/runner` 引入安全事件构建与 callback 分发逻辑，接入 Run/Stream 对应节点。
3. 在 `runtime/diagnostics` 与 `observability/event` 增加 S3 事件字段映射（additive）。
4. 增加安全事件契约测试（deny 告警、severity、Run/Stream 等价、invalid reload rollback）。
5. 增加 `scripts/check-security-event-contract.{sh,ps1}` 与 CI job `security-event-gate`。
6. 更新文档（配置示例、字段语义、告警契约、required-check 使用方式）。

## Open Questions

- 当前无阻断级开放问题；Webhook/多 sink、match 告警策略留待后续里程碑。
