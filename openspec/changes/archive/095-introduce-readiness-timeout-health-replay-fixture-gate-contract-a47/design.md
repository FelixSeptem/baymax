## Context

A40-A46 已将 readiness、admission、timeout resolution、adapter health governance 分别收敛为可测试契约，但这些能力目前主要按单域验证。当前缺口是“跨域交叉语义”的长期稳定回归基线：当多个域在一次运行中同时发生时（例如 `degraded + timeout clamp + half_open`），缺少统一 fixture 与断言规范。

A47 目标是在不引入平台化控制面的前提下，建立可版本化、可回放、可阻断的组合语义 gate，作为 0.x 收敛期的主线回归护栏。

## Goals / Non-Goals

**Goals:**
- 定义 readiness-timeout-health 组合 replay fixture 的最小覆盖矩阵与版本化规则。
- 固化 replay 输出断言模型（分类字段、trace 字段、计数字段）与 deterministic ordering。
- 建立 replay idempotency 校验，保证重复回放不膨胀逻辑聚合。
- 将组合回放套件纳入 `check-quality-gate.*` required checks，保持 shell/PowerShell parity。
- 约束 drift taxonomy：发现非 canonical finding/reason/trace 漂移时 fail-fast。

**Non-Goals:**
- 不引入新的 runtime 执行语义（不改 Run/Stream 终态机）。
- 不引入外部控制面、托管回放服务或远程状态仓库。
- 不替代各子能力既有单域契约，仅补齐交叉组合层。

## Decisions

### Decision 1: Fixture 采用“场景矩阵 + 单一权威输出”

- 方案：每个场景由 `input fixture + expected normalized output + assertion profile` 组成。
- 最小矩阵覆盖：
  - readiness: `ready|degraded|blocked` 与 strict/non-strict；
  - timeout: `profile/domain/request` precedence 与 parent clamp/reject；
  - health: `healthy|degraded|unavailable` + circuit `closed|open|half_open`；
  - path parity: Run/Stream。
- 原因：统一比较维度，避免各模块独立 fixture 造成口径分裂。

### Decision 2: Drift 检测默认阻断（fail-fast）

- 方案：当 expected 与 actual 在 canonical 字段上不一致时直接阻断 gate。
- 可容忍项：仅允许文档声明的 additive nullable 字段在“缺失->新增”窗口内不阻断。
- 原因：A47 是收敛提案，默认阻断比告警更符合目标。

### Decision 3: Replay 输出以语义断言优先，而非原始 JSON 全量字节比较

- 方案：
  - 强约束字段：status/code/reason/taxonomy/timeout_source/trace 关键节点/circuit_state；
  - 弱约束字段：时间戳、排序无关扩展字段采用规范化比较。
- 原因：减少非语义噪声导致的无效失败，同时保持关键语义不可漂移。

### Decision 4: Fixture 版本化与变更治理绑定 OpenSpec

- 方案：fixture schema 变更必须伴随 OpenSpec 变更；新增场景保持向后 additive。
- 原因：把 fixture 本身纳入 contract-first 治理，避免“测试脚本暗改语义”。

## Risks / Trade-offs

- [Risk] 夹具矩阵过大导致维护成本上升
  -> Mitigation: 固定最小必选矩阵 + 可选扩展矩阵分层维护。

- [Risk] 断言过严导致开发期频繁阻断
  -> Mitigation: 区分强约束/弱约束字段，并在文档明确兼容窗口。

- [Risk] 各域字段命名历史包袱导致规范化映射复杂
  -> Mitigation: 先冻结 canonical 映射表并在 `docs/mainline-contract-test-index.md` 对应到具体用例。

## Migration Plan

1. 在 `tool/diagnosticsreplay` 增加 A47 fixture loader 与 normalized assertion engine。
2. 在 `integration` 增加 readiness-timeout-health 交叉回放 suites 与 Run/Stream parity 验证。
3. 在 gate 脚本接入 A47 suites（shell/PowerShell 同步）。
4. 更新 contract index 与 runtime diagnostics 文档中的 fixture/field 对照表。
5. 在 README/roadmap 同步 A47 状态与验收口径。

## Open Questions

- 是否在 A47 首版纳入 “govulncheck 网络不可达” 场景的 fixture 说明（建议不纳入，留在环境治理层）。
- 是否需要将 A47 fixture 引擎独立成通用子包供后续 A48+ 复用（建议先内聚在 `tool/diagnosticsreplay`，后续再抽离）。
