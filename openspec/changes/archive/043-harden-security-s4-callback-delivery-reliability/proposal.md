## Why

S3 已提供 deny-only callback 告警能力，但当前告警投递仍是同步直调、无重试/限流/熔断治理，生产环境下容易因回调慢、回调故障或告警风暴放大主流程抖动。需要在不改变现有安全决策语义（deny 仍 deny）的前提下，补齐 S4 告警投递可靠性治理闭环。

## What Changes

- 新增 S4 告警投递治理：支持 `async` 默认投递模式、有界队列、溢出 `drop_old`、超时与最多 3 次重试策略。
- 新增断路器治理：引入 Hystrix 风格 `closed/open/half_open` 状态机与恢复窗口，降低持续失败时的放大效应。
- 扩展运行时配置：增加 `security.security_event.delivery.*` 配置域，并保持 `env > file > default` 与热更新回滚语义。
- 扩展诊断字段：增加投递模式、重试次数、队列丢弃、断路器状态/打开原因等增量字段。
- 增加独立门禁：新增 `security-delivery-gate` 契约测试作业（required-check 候选），覆盖 deny 触发、异步投递、重试、熔断、Run/Stream 语义等价。
- 保持边界不变：不新增外部 sink，不修改 `event_id` 生成策略，不改变 deny-only 触发规则。

## Capabilities

### New Capabilities
- `security-alert-delivery-governance-s4`: 定义 callback 告警投递可靠性治理（异步队列、重试、限流、断路器）与语义契约。

### Modified Capabilities
- `security-event-governance-s3`: 在 deny-only callback 规则上补充投递可靠性行为与失败退化语义。
- `runtime-config-and-diagnostics-api`: 扩展 S4 投递配置字段与增量诊断字段，补充热更新 fail-fast/rollback 约束。
- `go-quality-gate`: 增加 `security-delivery-gate` 独立 CI 门禁与脚本约束。

## Impact

- 受影响代码：`core/runner/*`、`runtime/config/*`、`runtime/diagnostics/*`、`observability/event/*`。
- 受影响测试：新增 S4 契约测试（async + drop_old + retry + circuit breaker + Run/Stream 等价 + invalid reload rollback）。
- 受影响 CI：新增 `security-delivery-gate` job 与跨平台校验脚本。
- 兼容性：pre-1.x 增量扩展；默认模式切换为 `async`，但保持安全决策结果与 deny 阻断语义不变。
