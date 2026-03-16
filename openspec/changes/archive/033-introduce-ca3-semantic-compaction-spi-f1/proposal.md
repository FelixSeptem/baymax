## Why

当前 CA3 已具备稳定的压力分区与降级链路，但 `squash/prune` 仍以工程剪裁为主（截断与规则删除），在高压场景下可能带来语义损耗。随着 Context Assembler 进入生产收敛阶段，需要在保持默认行为稳定的前提下，引入可演进的语义压缩策略能力。

## What Changes

- 在 `context/assembler` 引入 CA3 compaction SPI（包内复用），支持 `truncate` 与 `semantic` 两类策略。
- 默认策略保持 `truncate`（向后兼容）；`semantic` 策略首期落地为可执行实现，并通过当前 LLM client 完成压缩。
- 增加“证据最小集保留”规则（关键词 + 最近窗口），确保关键上下文在 `danger/emergency` 下不被误删。
- 增加 CA3 compaction 最小诊断字段：
  - `ca3_compaction_mode`
  - `ca3_compaction_fallback`
  - `ca3_compaction_retained_evidence_count`
- 统一 `Run/Stream` 在 compaction 决策与诊断输出上的语义一致性。
- 保持 `fail_fast/best_effort` 语义边界：`best_effort` 可回退，`fail_fast` 直接终止。
- 同步更新 README/docs，并将本期未实现的 semantic 质量增强 TODO 规划到 roadmap。

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `context-assembler-memory-pressure-control`: 增加语义压缩 SPI、证据最小集保留规则，以及 `Run/Stream` 一致性约束。
- `runtime-config-and-diagnostics-api`: 增加 CA3 compaction 配置与诊断字段契约，明确回退和 fail-fast 语义。

## Impact

- Affected code:
  - `context/assembler`（CA3 compaction SPI + semantic strategy + evidence retention）
  - `core/runner`（Run/Stream 透传与一致性验证）
  - `runtime/config`（新增 compaction 配置与校验）
  - `runtime/diagnostics`（新增 CA3 compaction 诊断字段）
- Affected docs:
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/context-assembler-phased-plan.md`
  - `docs/development-roadmap.md`
  - `docs/v1-acceptance.md`
  - `docs/mainline-contract-test-index.md`
- Compatibility:
  - 默认行为保持兼容（`truncate` 默认不变）。
  - 新字段为增量扩展，不破坏现有消费者。
