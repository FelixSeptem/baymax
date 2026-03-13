## Context

CA3 已提供分区控制、双阈值、spill/swap 与基础诊断字段，但当前仍存在三类生产问题：
1) 阈值策略计算在 stage override 与双触发场景的口径没有单一事实源；
2) token 计数链路虽已支持 provider 注入 + 本地预估，但回退语义与可观测口径需固定为稳定契约；
3) CA3/CA4 相关性能回归尚未形成明确 benchmark 门禁约束。

本设计聚焦 CA4“生产收敛”而非扩容新能力，确保后续 HITL/A2A/更复杂 memory 功能建立在稳定基线之上。

## Goals / Non-Goals

**Goals:**
- 固化阈值策略计算规则（global/stage 优先级、percent/absolute 触发解释、冲突选择）。
- 固化 token 计数回退语义（provider -> tiktoken -> lightweight estimate），并保证主流程不中断。
- 增强 Run/Stream 契约测试，确保 CA4 策略语义等价。
- 将 CA4 benchmark 纳入相对百分比门禁（含 P95）。
- 文档与实现严格对齐。

**Non-Goals:**
- 不实现 `db/object` spill backend。
- 不引入新的 context stage 或 agentic routing 行为。
- 不新增对外 API，仅在现有配置与诊断契约内收敛语义。

## Decisions

### 1) 阈值计算采用“阶段覆盖 + 双触发择高”
- 决策：先应用 stage override（若配置有效），再并行计算 percent/absolute 分区，最终取更高压力区。
- 理由：保持 deterministic 且可解释，避免不同阶段出现策略漂移。
- 备选：percent 与 absolute 设为主/备。
  - 放弃原因：会损失一类阈值的保护能力。

### 2) token 计数固定三层回退，不阻断主流程
- 决策：`sdk_preferred` 下：provider count 失败 -> tiktoken 估算失败 -> lightweight estimate；计数失败本身不触发 run fail-fast。
- 理由：计数属于策略信号，不应破坏主业务执行；同时保证离线/限网环境可运行。
- 备选：计数失败即中断。
  - 放弃原因：在受限环境会造成不必要可用性损失。

### 3) OpenAI 计数语义定位为“阈值策略估算”
- 决策：文档明确 OpenAI 计数用于 context assembler 阈值策略，不承诺账单精度语义。
- 理由：与项目定位一致，避免误导消费者。
- 备选：模拟账单级精确语义。
  - 放弃原因：成本高且受服务端隐式处理影响，难保证稳定。

### 4) 性能门禁纳入 CA4 benchmark
- 决策：新增 CA4 关键 benchmark 并以相对百分比阈值 + P95 作为门禁。
- 理由：CA4 目标是生产收敛，必须覆盖性能回归风险。
- 备选：仅保留功能测试。
  - 放弃原因：无法防止策略调整导致的吞吐/延迟回退。

## Risks / Trade-offs

- [Risk] tiktoken 在首次初始化需要词表资源，离线环境可能失败。
  → Mitigation: 固定 fallback 机制并在诊断中暴露计数来源。
- [Risk] 阈值规则更严格后，可能改变部分边界场景的历史行为。
  → Mitigation: 用契约测试覆盖边界案例并在文档中明确迁移预期。
- [Risk] benchmark 门禁提高 CI 成本。
  → Mitigation: 保持最小关键集，并复用现有性能策略脚本。

## Migration Plan

1. 固化阈值策略计算与回退语义实现。
2. 补齐 CA4 契约测试（Run/Stream + threshold + fallback）。
3. 新增/更新 benchmark 与门禁脚本接入。
4. 更新 README/docs 与 roadmap 对齐。
5. 通过全量质量门禁后归档。

## Open Questions

- 后续是否将 tiktoken 词表资源做本地可选打包（避免首启网络依赖），本期仅留 TODO。

## Implementation Closure Notes

- 发现：
  - stage override 在部分配置场景下缺少完整校验，可能导致阈值漂移。
  - Run/Stream 对 CA3 压力语义仅校验 zone，缺少 reason/trigger 等价约束。
  - 质量门禁缺少 CA4 专项 benchmark 回归检查。
- 修复：
  - 增加 stage override 边界校验与单元测试（完整配置通过，部分配置 fail-fast）。
  - 补齐 `ca3_pressure_trigger` 字段并打通 run payload/recorder/diagnostics。
  - 固化 `sdk_preferred` 计数回退语义并补齐 fallback 测试。
  - 新增 `BenchmarkCA4PressureEvaluation`（含 `p95-ns/op` 指标）与门禁脚本接入。
- 验证：
  - 单测：`context/assembler`、`core/runner`、`runtime/config`、`observability/event`。
  - 质量门禁：`go test` / `go test -race` / `golangci-lint` / `govulncheck` / CA4 benchmark regression。
