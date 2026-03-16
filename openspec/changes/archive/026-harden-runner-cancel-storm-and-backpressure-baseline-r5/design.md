## Context

当前仓库已完成多 Provider、Context Assembler（CA1-CA4）、Action Timeline（H1/H1.5）与 HITL（H2-H4）主干能力，主路径复杂度显著提升。现阶段在高并发 fanout 与取消风暴（上游超时、用户取消、下游阻塞）场景下，仍缺少统一、可验证的收敛基线。

现有 `runtime-concurrency-control` 已定义并发与背压语义，但尚未明确以下工程化约束：
- 默认背压策略如何选型并稳定落地；
- Run/Stream 在取消风暴下的语义一致性；
- tool/mcp/skill 三条执行路径的取消传播最小契约；
- 面向运维的最小诊断字段与性能门禁（含 p95 与 goroutine 峰值）。

本提案在不新增公开 API 的前提下，收敛运行时并发控制与观测契约，作为后续 A2A/多 Agent/Workflow 演进前的稳定性基线。

## Goals / Non-Goals

**Goals:**
- 在 `core/runner` 建立取消风暴与背压行为的统一实现基线。
- 默认背压策略固定为 `block`，保障主干语义稳定与 fail-fast 一致性。
- 通过 runtime 配置暴露并发控制参数，不新增公共 API surface。
- Run/Stream 在等价场景下保持取消传播语义一致。
- 将 `tool/mcp/skill` 三条路径纳入取消传播契约测试。
- 新增最小诊断字段：`cancel_propagated_count`、`backpressure_drop_count`、`inflight_peak`。
- 建立性能验收口径：`p95 latency` 与 `goroutine peak`。

**Non-Goals:**
- 不引入新的外部协议能力（如 A2A 实现）。
- 不新增 CLI 调试面，仅保持库接口路径。
- 不在本期实现 `drop_low_priority` 策略，仅保留配置与实现扩展 TODO。
- 不调整既有业务能力（CA/HITL/provider）的语义边界。

## Decisions

### Decision 1: 背压默认策略固定为 `block`
- 决策：默认背压采用 `block`，当达到并发上限/队列阈值时阻塞新任务进入，直到可用窗口恢复。
- 原因：
  - 与现有 fail-fast + 语义稳定原则一致；
  - 可避免“静默丢弃”导致的行为不可解释；
  - 更利于建立首版可观测基线和性能阈值。
- 备选方案：`drop_low_priority`。
  - 未采用原因：当前尚无统一优先级模型，直接丢弃会引入语义不确定性。
  - 后续：保留 TODO，待优先级模型成熟后增量接入。

### Decision 2: 配置入口复用 `runtime/config`，不新增公开 API
- 决策：并发与背压新字段进入既有 runtime 配置层，沿用 `env > file > default` 与 fail-fast 校验链路。
- 原因：
  - 保持项目 library-first 定位；
  - 降低接入心智负担，避免并发控制出现旁路配置入口。
- 备选方案：新增 runner options/public API。
  - 未采用原因：会扩大对外面，增加版本兼容成本。

### Decision 3: 取消传播契约统一覆盖 `tool/mcp/skill`
- 决策：定义“父上下文取消后，子任务不得再接收新工作，已在途工作在超时策略内收敛”的统一行为，并对 Run/Stream 双路径做契约测试。
- 原因：
  - 三条路径是当前主干真实高并发风险点；
  - 能直接减少 goroutine 残留与事件漂移。
- 备选方案：只覆盖 tool。
  - 未采用原因：mcp/skill 会成为未覆盖盲区，导致测试信号失真。

### Decision 4: 观测最小化扩展 + 明确性能门禁
- 决策：仅新增三个诊断字段，并在测试/benchmark 验收中强制输出 `p95 latency` 与 `goroutine peak`。
- 原因：
  - 控制变更面；
  - 保证可运维性指标可比较、可回归。
- 备选方案：一次性扩展更多聚合字段。
  - 未采用原因：会增加实现复杂度与文档负担，不利于快速稳定落地。

## Risks / Trade-offs

- [Risk] `block` 策略在极端流量下可能增加排队延迟。  
  → Mitigation: 增加阈值配置、p95 监控与超时保护，避免无限等待。

- [Risk] Run/Stream 双路径实现细节不同，可能出现取消语义漂移。  
  → Mitigation: 增加等价契约测试矩阵，并将失败路径纳入主干 CI。

- [Risk] 将 `tool/mcp/skill` 全覆盖会提高测试复杂度与耗时。  
  → Mitigation: 分层测试（单元 + 集成 + 压测样例），并限制每层最小必测场景。

- [Risk] 新增诊断字段如果映射不完整，会造成可观测数据不一致。  
  → Mitigation: 为 recorder/store 增加字段稳定性测试，确保 run.finished 到 diagnostics 的映射闭环。

## Migration Plan

1. 在 `runtime/config` 添加并发与背压字段，并补齐默认值与 fail-fast 校验。  
2. 在 `core/runner` 接入统一限流/背压入口，默认 `block`。  
3. 将取消传播逻辑统一下沉到 runner 执行路径，并覆盖 `tool/mcp/skill`。  
4. 扩展 `runtime/diagnostics` 与 `observability/event` 字段映射。  
5. 补齐 Run/Stream 契约测试与压力场景测试（含 `p95`、`goroutine peak`）。  
6. 同步 README/docs 与 roadmap，更新主干契约索引。  
7. 回归验证：`go test ./...`、`go test -race ./...`、`golangci-lint`。

回滚策略：若出现性能或语义回归，关闭新增并发配置项并退回默认策略；保留旧配置解析与事件结构兼容。

## Open Questions

- `drop_low_priority` 的优先级来源是静态规则还是动态打分（本期仅记录 TODO，不实现）。
- `goroutine peak` 统计口径是否长期保持“run 级”还是增加“phase 级”细分（本期以 run 级为准）。
