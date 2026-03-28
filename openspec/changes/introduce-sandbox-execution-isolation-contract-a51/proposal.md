## Why

当前 S2/S3/S4 已覆盖权限、限流、IO 过滤与 deny 告警投递，但高风险工具执行仍主要依赖 in-process 路径。对于 shell/文件系统/进程类调用，仅靠策略判定无法提供执行隔离，需要补齐可观测、可回放、可回滚的 sandbox 运行契约。

## What Changes

- 新增 sandbox execution isolation 契约，定义工具调用 `host|sandbox|deny` 决策与执行语义。
- 新增宿主注入式 `SandboxExecutor` 接缝（lib-first），统一接入 `tool/local` 与 `mcp/stdio` 启动路径。
- 冻结 sandbox 执行输入契约（ExecSpec）与执行结果契约（ExecResult），统一承载 command/env/mount/network/resource/session 等跨 sandbox 后端共性参数。
- 新增 executor capability negotiation 契约，支持声明 `required_capabilities` 并在 startup/readiness/admission 路径 fail-fast 阻断不满足能力。
- 冻结 sandbox backend/capability canonical 枚举与字段级 schema（单位、边界、默认）以保障跨后端可互换性。
- 新增 `security.sandbox.*` 配置域，纳入 `env > file > default`、启动 fail-fast、热更新原子回滚。
- 新增 `observe|enforce` 双模式及 `required/fallback_action` 收敛策略，支持渐进启用与快速回滚。
- 对高风险工具冻结 deny-first fallback 默认策略，`allow_and_record` 仅允许显式 per-selector override。
- 扩展 readiness preflight + admission guard，在 `sandbox.required=true` 且执行器不可用时支持 deterministic deny。
- 一次性补齐 sandbox 可观测性闭环：
  - Action Timeline reason taxonomy（sandbox path/deny/fallback/timeout）
  - Runtime diagnostics additive 字段与 single-writer idempotency
  - S3/S4 安全事件与告警投递语义（deny-only + delivery governance）
  - Replay fixture drift 分类与质量门禁阻断映射
- 在质量门禁中新增 sandbox contract 检查脚本，并保持 shell/PowerShell parity。
- 在 A51 首期同步交付 sandbox executor conformance harness（offline deterministic）并纳入 gate 阻断。
- 本提案目标是一次性冻结 sandbox 接入与观测 contract，后续仅做同 contract 下实现扩展，不再拆 sandbox 语义提案。

## Capabilities

### New Capabilities
- `sandbox-execution-isolation`: 定义沙箱决策、执行、回退与 Run/Stream 等价语义。

### Modified Capabilities
- `tool-security-governance-s2`: 增加 tool governance 与 sandbox 决策组合语义及 fallback 行为约束。
- `runtime-config-and-diagnostics-api`: 增加 `security.sandbox.*` 配置与 sandbox additive 诊断字段。
- `runtime-readiness-preflight-contract`: 增加 sandbox required 可用性预检与 canonical finding。
- `runtime-readiness-admission-guard-contract`: 增加 sandbox required 场景的 admission fail-fast 契约。
- `action-timeline-events`: 增加 sandbox reason taxonomy 与 Run/Stream timeline 等价约束。
- `diagnostics-single-writer-idempotency`: 增加 sandbox 诊断写入幂等与重放去重约束。
- `security-event-governance-s3`: 增加 sandbox deny 事件 taxonomy 与 deny-only 告警语义。
- `security-alert-delivery-governance-s4`: 增加 sandbox deny 告警在 async/retry/circuit 下的投递治理契约。
- `diagnostics-replay-tooling`: 增加 sandbox fixture 与 drift 分类断言。
- `diagnostics-query-performance-baseline`: 增加 sandbox 字段场景下 QueryRuns 回归基线约束。
- `go-quality-gate`: 增加 sandbox contract gate 的阻断映射。

## Impact

- 代码：
  - `core/runner`（sandbox 决策与执行路径）
  - `tool/local`（高风险工具 sandbox 分发）
  - `mcp/stdio`（命令启动沙箱化接缝）
  - `core/types`（ExecSpec/ExecResult/SandboxExecutor SPI）
  - `runtime/config`、`runtime/config/readiness*`（配置、校验、准入）
  - `runtime/diagnostics`、`observability/event`（additive 字段、single-writer、幂等写入）
  - `action.timeline` 事件发射与聚合（sandbox reason taxonomy）
  - `core/runner/security_delivery`（sandbox deny 告警投递治理复用）
  - `tool/diagnosticsreplay`、`integration/*`（fixture、drift guard、Run/Stream parity）
  - `scripts/check-quality-gate.*` 与新增 sandbox gate 脚本
- 文档：
  - `docs/runtime-config-diagnostics.md`
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
  - `README.md`
- 兼容性：
  - 默认保持 `security.sandbox.enabled=false`，不改变现有主路径。
  - 通过 `observe -> enforce` 渐进启用，异常时可快速回退到 host 模式。
