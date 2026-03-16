## Why

当前 `skill/loader` 已有轻量语义触发逻辑，但评分规则、同分决策、低置信度抑制与配置来源尚未形成稳定契约，导致行为可预期性与回归可测性不足。随着 R3/R4 的技能生态扩展，这一能力需要先收敛为可配置、可测试、可演进的基线，并为后续 embedding 评分器预留扩展接口。

## What Changes

- 在 `skill/loader` 收敛可插拔评分接口，默认策略固定为“关键词加权 + 阈值命中”，本期不引入 embedding 实现。
- 新增同分决策规则：默认采用 `highest-priority`，保证候选技能选择稳定且可解释。
- 新增“低置信度不触发”默认安全策略（开启），避免弱匹配误触发技能。
- 将触发评分策略参数暴露到 runtime YAML（保持 `env > file > default`），并补齐 fail-fast 校验。
- 增加主干契约测试与回归用例，覆盖 Run/Stream 语义不变前提下的触发一致性、配置生效与边界行为。
- 为 embedding 评分器保留 internal 接口与 TODO，不新增公开 API。

## Capabilities

### New Capabilities
- `skill-trigger-scoring`: 定义技能语义触发评分、阈值命中、同分策略与低置信度抑制的标准行为契约。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加技能触发评分策略的 runtime YAML 配置项、优先级与校验约束。

## Impact

- Affected code:
  - `skill/loader/*`
  - `runtime/config/*`
  - `core/types/*`（仅内部扩展接口/枚举）
  - `integration/*` 或 `core/runner/*`（契约测试）
- Affected docs:
  - `README.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/development-roadmap.md`
  - `docs/v1-acceptance.md`
  - `docs/mainline-contract-test-index.md`
- API impact: 不新增对外公开 API；评分器扩展点仅包内/内部使用。
