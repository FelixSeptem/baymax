## Why

在 `033-introduce-ca3-semantic-compaction-spi-f1` 完成 CA3 compaction SPI 与 semantic 路径后，语义压缩仍缺少可配置质量门控与模板安全控制。当前语义路径在高压场景下可运行，但需要进一步提高可观测性和可控性，避免低质量压缩结果进入模型执行路径。

本变更聚焦 F2 收敛：
- 增加规则化质量评分与阈值门控；
- 增加 runtime 可配置语义模板白名单控制；
- 保留 embedding scorer 的 SPI hook（不绑定 provider adapter）；
- 扩展诊断字段以支持故障与质量回溯；
- 补齐 benchmark 基线与 Run/Stream 契约一致性验证。

> 注：该归档目录原始文件在归档异常中缺失；当前文档依据仓库实现与文档事实恢复重建（Recovered）。

## What Changes

- 在 CA3 semantic compaction 增加质量门控：`coverage/compression/validity` 规则评分 + `threshold` 判定。
- 质量不达标时：
  - `best_effort`：回退 `truncate` 并记录回退原因；
  - `fail_fast`：终止 assembly。
- 增加 runtime 模板控制：`semantic_template.prompt` + `allowed_placeholders`，并在启动/热更新做 fail-fast 校验。
- 增加 embedding SPI hook 配置位：`embedding.enabled + selector`，本期不绑定具体 adapter。
- 扩展 run diagnostics 字段：
  - `ca3_compaction_fallback_reason`
  - `ca3_compaction_quality_score`
  - `ca3_compaction_quality_reason`
- 同步 benchmark 与文档，保证 Run/Stream 语义等价。

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `context-assembler-memory-pressure-control`
- `runtime-config-and-diagnostics-api`

## Impact

- Affected code:
  - `context/assembler`
  - `runtime/config`
  - `core/runner`
  - `runtime/diagnostics`
  - `observability/event`
  - `integration/*benchmark*`
- Affected docs:
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/context-assembler-phased-plan.md`
  - `docs/development-roadmap.md`
  - `docs/v1-acceptance.md`
- Compatibility:
  - 默认行为保持兼容（默认 `truncate`）。
  - 新增字段均为增量扩展，不破坏既有消费者。
