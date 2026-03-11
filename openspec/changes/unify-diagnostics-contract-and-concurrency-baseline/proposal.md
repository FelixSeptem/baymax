## Why

当前诊断数据存在“业务直写 + 事件录制”双通路并存，run/skill 指标在并发与重试路径下有重复记录风险，影响诊断 API 的统计可信度。将单写入、契约一致性与并发安全基线合并为一次迭代，可在较小范围内一次性消除数据口径漂移并提升后续扩展稳定性。

## What Changes

- 新增诊断写入治理能力：确立单一写入通路（single-writer），禁止同语义事件双写。
- 新增诊断幂等机制：为 run/skill 诊断引入稳定幂等键并在存储写入层执行去重策略。
- 加固诊断契约：统一 run/skill 事件 schema、状态枚举与错误语义，补齐契约测试覆盖（success/failure/warning/retry）。
- 将并发安全纳入质量基线：默认执行 `go test -race ./...`，并为诊断聚合和并发写入路径补充 race/并发单测。
- 调整文档与变更说明：更新 README 与 docs 的诊断链路、幂等策略、质量门禁说明。

## Capabilities

### New Capabilities
- `diagnostics-single-writer-idempotency`: 统一诊断写入通路并提供 run/skill 级别的幂等写入保证。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 诊断 API 的统计口径改为单写入 + 幂等语义，并明确 schema/错误语义约束。
- `go-quality-gate`: 将 race 检测与关键并发测试纳入默认质量门禁要求。

## Impact

- 代码目录：`core/runner`、`skill/loader`、`observability/event`、`runtime/diagnostics`（或等价诊断存储/接口目录）。
- 测试影响：新增并发写入、重复事件注入、重试重放场景测试；更新质量门禁脚本与 CI 配置。
- API 影响：诊断 API 对外字段保持语义一致，但数据来源与聚合逻辑收敛为单写入语义。
- 运维影响：诊断指标更稳定，误报与重复样本降低，故障定位可解释性提升。