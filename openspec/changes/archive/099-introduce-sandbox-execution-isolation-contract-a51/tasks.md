## 1. Config and Policy Surface

- [x] 1.1 在 `runtime/config` 增加 `security.sandbox.*` 配置结构、默认值与 `env > file > default` 解析。
- [x] 1.2 增加 `mode/default_action/fallback_action/profile/by_tool` 启动校验与热更新原子回滚测试。
- [x] 1.3 增加 executor/capability 配置面（`backend/session_mode/required_capabilities`）与 fail-fast 校验。
- [x] 1.4 冻结 backend/capability canonical 枚举并补齐非法枚举 fail-fast 测试。
- [x] 1.5 冻结 ExecSpec 字段级单位/边界/默认值并补齐 schema 校验测试。
- [x] 1.6 在 `core/types` 与 `core/runner` 增加 sandbox action resolve 与标准错误/reason code 映射。

## 2. Sandbox Executor Integration

- [x] 2.1 定义宿主注入式 `SandboxExecutor` SPI，并在 runner options 中提供接入点。
- [x] 2.2 冻结 `ExecSpec/ExecResult` 字段并补齐跨后端标准化映射（command/env/mount/network/resource/session/timeouts）。
- [x] 2.3 在 `tool/local` 高风险工具路径接入 `host|sandbox|deny` 执行分支与 fallback 行为。
- [x] 2.4 为 in-process 工具增加 sandbox adapter bridge；未适配工具在 observe/enforce 下保持 deterministic 行为。
- [x] 2.5 在 `tool/local`/`runtime/config` 实现高风险 selector deny-first fallback 默认语义，allow fallback 需显式 override。
- [x] 2.6 在 `mcp/stdio` 命令启动路径接入 sandbox launcher 分支并实现 `per_call|per_session` 生命周期语义。
- [x] 2.7 补齐 `mcp/stdio` sandbox 生命周期异常测试（session crash、cancel、reconnect、close idempotent）。

## 3. Readiness and Admission Guard

- [x] 3.1 在 `runtime/config/readiness` 增加 `sandbox.required` 可用性预检 finding 与 strict/non-strict 分类测试。
- [x] 3.2 在 managed admission 路径接入 sandbox required deny 语义，并验证 deny path side-effect free。
- [x] 3.3 补齐 Run/Stream 等价测试，确保 sandbox 相关 primary/secondary explainability 字段一致。

## 4. Observability Contract Closure (一次性收口)

- [x] 4.1 在 `core/runner` timeline 发射路径增加 sandbox canonical reasons（deny/launch_failed/timeout/fallback）。
- [x] 4.2 在 `runtime/diagnostics` 与 `observability/event.RuntimeRecorder` 增加 sandbox additive 字段并保持 single-writer idempotency（含 backend/capability/resource/latency 字段）。
- [x] 4.3 在 `core/runner/security*` 与 S3/S4 路径接入 sandbox deny 事件 taxonomy 与 delivery 语义（queue/retry/circuit）。
- [x] 4.4 在 `tool/diagnosticsreplay` 增加 sandbox fixture（`a51.v1`）与 drift 分类断言（policy/fallback/timeout/capability/resource/session）。
- [x] 4.5 在 `integration` 增加 Run/Stream parity + replay idempotency + S3/S4 delivery parity + capability negotiation 套件。

## 5. Quality Gate and Performance Baseline

- [x] 5.1 新增 `check-security-sandbox-contract.sh/.ps1` 并接入 `check-quality-gate.*`。
- [x] 5.2 在 diagnostics query benchmark 基线中加入 sandbox-enriched 数据集覆盖与阈值回归断言。
- [x] 5.3 增加 backend compatibility matrix smoke suites（至少 Linux sandbox path + container/job path）并接入 sandbox gate。
- [x] 5.4 增加 sandbox executor conformance harness suites（offline deterministic）并接入 sandbox gate。
- [x] 5.5 将 sandbox gate 暴露为独立 required-check 候选并同步 shell/PowerShell parity。

## 6. Docs and Validation

- [x] 6.1 更新 `docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`、`README.md`。
- [x] 6.2 执行并记录最小验收命令：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`。
