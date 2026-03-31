## 1. Runner ReAct Loop Core

- [x] 1.1 在 `core/runner` 抽象 Run/Stream 共享 ReAct loop 核心（model step、tool dispatch、tool result feedback、termination）。
- [x] 1.2 固化 step-boundary 执行顺序，保证 Stream 保持增量输出同时支持 step 结束后的工具分发与回灌。
- [x] 1.3 实现 canonical ReAct 终止 taxonomy（`react.completed`、预算耗尽、dispatch 失败、provider 错误、取消）。
- [x] 1.4 补齐 ReAct loop 核心单测（确定性终止、错误路径、取消路径、重复事件幂等）。

## 2. Stream Tool Dispatch Parity

- [x] 2.1 在 Stream 主路径补齐工具分发与结果回灌，移除 `stream_tool_dispatch_not_supported` 中间态语义。
- [x] 2.2 对齐 Run/Stream 在 ReAct 场景下的语义等价断言（终止 reason、loop counters、budget hit）。
- [x] 2.3 补齐 Run/Stream ReAct parity integration tests（成功、预算耗尽、tool 失败、provider 错误）。

## 3. ReAct Budget Governance

- [x] 3.1 新增 run-level `tool_call_limit` 与现有 iteration-level 上限协同治理。
- [x] 3.2 实现预算触发 fail-fast 语义（无额外 tool dispatch、无额外 model step）。
- [x] 3.3 补齐预算治理测试（`max_iterations`、`tool_call_limit`、双阈值同时逼近的确定性裁决）。

## 4. Multi-Provider Tool-Calling Canonicalization

- [x] 4.1 在 `model/openai`、`model/anthropic`、`model/gemini` 统一 tool-call request canonical 映射。
- [x] 4.2 统一 tool-result feedback canonical 映射与 step correlation 规则。
- [x] 4.3 收敛 provider tool-calling error taxonomy（capability unsupported、request invalid、feedback invalid、rate/auth）。
- [x] 4.4 补齐 provider 归一测试与 step-boundary fallback 测试（禁止 mid-step provider switch）。

## 5. Runtime Config and Diagnostics Integration

- [x] 5.1 在 `runtime/config` 新增 `runtime.react.*` 字段、默认值、`env > file > default` 解析与校验。
- [x] 5.2 实现 ReAct 配置热更新原子切换与非法更新回滚。
- [x] 5.3 在 `runtime/diagnostics` 与 `observability/event.RuntimeRecorder` 增加 ReAct additive 字段并保持 bounded-cardinality。
- [x] 5.4 补齐 ReAct 配置与诊断单测（启动 fail-fast、热更新回滚、single-writer idempotency）。

## 6. Readiness and Admission Closure

- [x] 6.1 在 `runtime/config/readiness` 增加 `react.*` canonical findings（loop、stream dispatch、provider、tool registry、sandbox dependency）。
- [x] 6.2 固化 strict/non-strict 映射与 deterministic finding taxonomy 断言。
- [x] 6.3 在 admission guard 增加 ReAct 前置阻断逻辑，并保持 deny side-effect-free。
- [x] 6.4 补齐 readiness + admission ReAct 集成测试（allow_and_record、fail_fast、Run/Stream 等价）。

## 7. Sandbox Consistency in ReAct Loops

- [x] 7.1 对齐 ReAct 多轮工具调用下 sandbox action resolution 一致性（host/sandbox/deny）。
- [x] 7.2 对齐 sandbox fallback（allow_and_record/deny）在 ReAct 迭代中的 taxonomy 与计数语义。
- [x] 7.3 补齐 sandbox capability mismatch 在 ReAct loop 中的终止语义测试与 Run/Stream parity 测试。

## 8. Replay Fixture and Drift Guard

- [x] 8.1 在 `tool/diagnosticsreplay` 新增 `react.v1` fixture schema、loader、normalization。
- [x] 8.2 新增 ReAct drift 分类断言（loop step、budget、termination、stream dispatch、provider mapping）。
- [x] 8.3 增加 mixed-fixture 回放兼容测试（历史 fixture + `react.v1` 同跑）。

## 9. Quality Gate and CI Wiring

- [x] 9.1 新增 `scripts/check-react-contract.sh/.ps1` 并接入 `check-quality-gate.*`。
- [x] 9.2 在 CI 暴露独立 required-check 候选（`react-contract-gate`）。
- [x] 9.3 校验 shell/PowerShell gate parity（失败传播、退出码、阻断语义一致）。

## 10. Docs, Examples, and One-Shot Closure

- [x] 10.1 更新 `README.md` 与 `examples`，提供 ReAct 最小接入蓝图（Run/Stream 等价、tool loop、配置示例）。
- [x] 10.2 更新 `docs/runtime-config-diagnostics.md`（`runtime.react.*` 配置与 ReAct additive 字段）。
- [x] 10.3 更新 `docs/mainline-contract-test-index.md` 与 `docs/development-roadmap.md`（A56 contract/replay/gate 映射）。
- [x] 10.4 执行一次性闭环审查：确认 ReAct 主题在 loop/provider/readiness/admission/sandbox/replay/gate/docs 全链路已覆盖，不留 ReAct 后续拆案的必需项。

## 11. Validation

- [x] 11.1 执行 `go test ./...`。
- [x] 11.2 执行 `go test -race ./...`。
- [x] 11.3 执行 `golangci-lint run --config .golangci.yml`。
- [x] 11.4 执行 `pwsh -File scripts/check-react-contract.ps1` 与 `pwsh -File scripts/check-quality-gate.ps1`。
- [x] 11.5 执行 `pwsh -File scripts/check-docs-consistency.ps1`。
