## Why

S2 已具备 `deny` 级别的安全阻断能力，但仍缺少统一的安全事件分类与告警契约，导致生产侧难以稳定聚合、分级和联动处置。现在需要补齐 S3 事件治理闭环，把阻断结果转化为可运营的标准化安全事件流。

## What Changes

- 新增 S3 安全事件治理能力：定义统一事件 taxonomy（事件类型、严重级别、标准 reason code）。
- 新增运行时告警契约：仅对 `deny` 决策触发告警，不对 `match/allow` 触发告警。
- 新增告警扩展接口：支持 host 注入 callback sink，作为首期唯一告警输出路径。
- 新增运行时配置字段：支持事件开关、deny 告警策略和 callback 注册约束，沿用 `env > file > default`。
- 新增 Run/Stream 等价约束：相同输入与配置下，安全事件与告警语义必须等价。
- 新增独立 CI 门禁：`security-event-gate` 作为 required-check 候选。

## Capabilities

### New Capabilities
- `security-event-governance-s3`: 定义安全事件 taxonomy、deny-only 告警触发规则、callback sink 契约与 Run/Stream 语义等价约束。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 扩展 S3 事件与告警配置字段、诊断记录字段和热更新回滚语义。
- `go-quality-gate`: 增加 `security-event-gate` 独立契约门禁并暴露为 required-check 候选。
- `tool-security-governance-s2`: 补充 deny 决策到 S3 安全事件映射与规范化 reason code 约束。
- `model-io-security-filtering`: 补充 deny 决策到 S3 安全事件映射与规范化 reason code 约束。

## Impact

- 受影响代码：`core/runner/*`、`runtime/config/*`、`runtime/diagnostics/*`、`observability/event/*`。
- 受影响测试：新增安全事件 contract tests（deny 触发、severity 分级、Run/Stream 等价、无效热更新回滚）。
- 受影响 CI：新增 `security-event-gate` job 与跨平台校验脚本。
- 兼容性：pre-1.x 增量扩展，不要求向后兼容承诺；默认行为保持不破坏现有执行路径。
