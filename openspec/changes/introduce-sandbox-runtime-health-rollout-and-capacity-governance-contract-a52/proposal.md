## Why

A51 已冻结 sandbox 执行隔离 contract，但实施落地仍缺少“灰度策略 + 健康度判定 + 容量治理 + 自动冻结”一体化运行治理语义。若没有统一 rollout contract，业务侧会以脚本或外置开关分散实现，导致 Run/Stream 语义漂移、回滚不可审计、跨后端行为不可回放。

## What Changes

- 新增 sandbox 运行治理能力，定义 rollout phase 状态机（`observe|canary|baseline|full|frozen`）及合法迁移规则。
- 新增 sandbox 健康预算 contract，冻结 canonical SLI/SLO 输入（超时率、启动失败率、违规率、P95 时延漂移、准入拒绝率）和 breach 判定语义。
- 新增 sandbox 容量治理 contract，冻结 admission 动作（`allow|throttle|deny`）与队列/并发预算语义。
- 新增自动冻结与受控解冻 contract（freeze reason taxonomy、cooldown、manual override token 语义）。
- 新增 `security.sandbox.rollout.*` 配置域并纳入 `env > file > default`、启动 fail-fast、热更新原子回滚。
- 扩展 readiness preflight + admission guard，将 rollout/freeze/capacity 作为执行前 deterministic 判定输入。
- 扩展 timeline/diagnostics/replay：增加 rollout/capacity/freeze additive 字段、reason taxonomy、drift 分类与回放夹具。
- 在质量门禁新增 rollout governance contract gate，并保持 shell/PowerShell parity 与独立 required-check 候选暴露。
- 本提案不改变 A51 的 ExecSpec/ExecResult 与 backend capability 基础 contract，仅补齐“上线运行治理”层。

## Capabilities

### New Capabilities
- `sandbox-runtime-rollout-and-capacity-governance`: 定义 sandbox 灰度发布、健康预算、容量治理与冻结/解冻统一契约。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 增加 rollout/capacity 配置与 additive 诊断字段语义。
- `runtime-readiness-preflight-contract`: 增加 rollout/freeze/health-budget 的 preflight finding 与 strict/non-strict 映射。
- `runtime-readiness-admission-guard-contract`: 增加 rollout capacity action 与 freeze fail-fast 准入语义。
- `action-timeline-events`: 增加 rollout/freeze/capacity 的 canonical reason taxonomy。
- `diagnostics-replay-tooling`: 增加 A52 rollout fixture 与 drift 分类断言。
- `diagnostics-query-performance-baseline`: 增加 sandbox-enriched 查询负载基线与回归阈值覆盖。
- `go-quality-gate`: 增加 rollout governance gate 与独立 required-check 暴露。

## Impact

- 代码：
  - `runtime/config`（`security.sandbox.rollout.*` 配置域、校验、热更新回滚）
  - `runtime/config/readiness*`（rollout/freeze/capacity finding 分类）
  - `core/runner` 与 admission 路径（capacity action 及 freeze deny 语义）
  - `runtime/diagnostics`、`observability/event`（rollout/capacity/freeze additive 字段）
  - `tool/diagnosticsreplay`、`integration/*`（A52 fixture、drift、Run/Stream parity）
  - `scripts/check-quality-gate.*` 与新增 rollout gate 脚本
- 文档：
  - `docs/development-roadmap.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `README.md`
- 兼容性：
  - 默认保持 `security.sandbox.rollout.phase=observe`，不改变 A51 默认主路径。
  - 仅新增 additive 字段与治理逻辑，不移除既有字段与错误分类。
