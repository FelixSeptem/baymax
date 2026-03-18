## Why

A5/A6 主功能实施后，仍需要一个独立收口迭代来稳定契约、门禁与兼容语义，否则容易出现“功能可用但回归不可控”的问题。A7 作为治理收尾层，聚焦 bounded-cardinality、reason taxonomy、shared-contract gate 扩展与 CI 契约门禁闭环。

## What Changes

- 为多代理组合链路新增 bounded-cardinality 约束，防止高并发/重放下诊断字段膨胀。
- 固化 A5/A6 新增字段的兼容窗口语义（additive/nullable/default），并补迁移口径。
- 固化 scheduler/subagent timeline reason taxonomy 与 attempt-level 关联字段要求。
- 扩展 shared-contract gate，纳入 scheduler/subagent 命名空间、关联字段与单写入口检查。
- 增加独立 scheduler crash-recovery/takeover 契约门禁套件并纳入质量门禁。
- 同步更新主干契约索引与相关文档，保证代码/测试/文档一致。

## Capabilities

### New Capabilities
- `multi-agent-tail-governance`: 定义 A5/A6 收口治理要求（契约收敛、门禁增强、兼容窗口与观测基线）。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 bounded-cardinality 与兼容窗口要求。
- `action-timeline-events`: 增加 scheduler/subagent canonical reason 与 attempt-level 关联字段要求。
- `runtime-module-boundaries`: 增加 scheduler/subagent 共享契约门禁检查项。
- `go-quality-gate`: 增加 scheduler crash-recovery/takeover 契约门禁要求。

## Impact

- 影响代码：`runtime/diagnostics/*`、`observability/event/*`、`tool/contributioncheck/*`、`scripts/check-*.ps1|sh`、`.github/workflows/*`。
- 影响测试：新增/更新 scheduler recovery、idempotency、Run/Stream 等价与 gate 检查测试。
- 影响文档：`docs/mainline-contract-test-index.md`、`docs/runtime-config-diagnostics.md`、`docs/runtime-module-boundaries.md`、`docs/v1-acceptance.md`、`docs/development-roadmap.md`。
- 兼容性：严格 additive，不移除既有字段；通过兼容窗口声明约束新增字段消费方式。
