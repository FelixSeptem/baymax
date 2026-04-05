## 1. Runtime JIT Config Schema and Validation

- [x] 1.1 在 `runtime/config` 增加 `runtime.context.jit.reference_first.*` 字段、默认值与 env 映射。
- [x] 1.2 在 `runtime/config` 增加 `runtime.context.jit.isolate_handoff.*` 字段、默认值与 env 映射。
- [x] 1.3 在 `runtime/config` 增加 `runtime.context.jit.edit_gate.*` 字段、默认值与 env 映射。
- [x] 1.4 在 `runtime/config` 增加 `runtime.context.jit.swap_back.*` 与 `runtime.context.jit.lifecycle_tiering.*` 字段、默认值与 env 映射。
- [x] 1.5 实现配置校验（枚举、阈值、组合约束）并保持 fail-fast。
- [x] 1.6 增加配置优先级测试（`env > file > default`）与热更新非法配置原子回滚测试。

## 2. Reference-First Stage2 Assembly

- [x] 2.1 在 `context/*` 定义 `discover_refs -> resolve_selected_refs` 两段式流程与 canonical payload。
- [x] 2.2 实现 “先引用后正文” 策略（引用优先注入、按需展开正文）。
- [x] 2.3 增加 reference 选择预算控制（最大引用数、展开 token 上限、缺失引用策略）。
- [x] 2.4 增加 schema 单测（缺字段、非法 locator、预算越界、重复引用去重）。

## 3. Isolate Handoff Contract

- [x] 3.1 定义子代理回传固定结构：`summary`、`artifacts[]`、`evidence_refs[]`、`confidence`、`ttl`。
- [x] 3.2 在主代理路径实现默认消费策略（优先摘要与引用，正文按策略延后解析）。
- [x] 3.3 增加 handoff 有效性校验（confidence 边界、ttl 过期、引用不存在）。
- [x] 3.4 增加 replay 幂等测试（重复 handoff 不膨胀计数且终态等价）。

## 4. Context Edit Gate

- [x] 4.1 实现 `clear_at_least` 阈值判定（estimated saved tokens 与稳定性收益比）。
- [x] 4.2 实现 edit gate 决策分流（通过/拒绝）并记录 canonical decision reason。
- [x] 4.3 增加 edit gate 负向测试（阈值过高、收益不足、配置冲突）。
- [x] 4.4 保持未达阈值时语义不变，不引入隐式清理动作。

## 5. Relevance Swap-Back and Lifecycle Tiering

- [x] 5.1 将 swap-back 逻辑升级为 query + evidence tag 相关性回填。
- [x] 5.2 实现 `hot|warm|cold` 生命周期分层及跨层迁移策略（write/compress/prune/spill）。
- [x] 5.3 增加 tiering 与 swap-back 组合测试（相关性阈值、TTL 过期、跨层回填）。
- [x] 5.4 增加与 A68 interrupt/resume 边界兼容测试（恢复游标边界内行为稳定）。

## 6. Task-Aware Recap

- [x] 6.1 将 tail recap 升级为 task-aware 结构化 recap（基于本轮选择/剪裁/外化动作）。
- [x] 6.2 增加 `context_recap_source` 来源标记与稳定序列化。
- [x] 6.3 增加 recap 语义回归测试（避免固定模板回落和无关摘要注入）。

## 7. Run/Stream Parity and Boundary Regression

- [x] 7.1 增加 equivalent Run/Stream parity 集成测试（reference-first、edit gate、swap-back、tiering、recap）。
- [x] 7.2 增加 A56 终止 taxonomy 与 A58 决策解释链边界回归测试（不引入平行语义）。
- [x] 7.3 增加 A57 安全治理回归测试（context 组织改动不绕过 sandbox/egress/allowlist）。
- [x] 7.4 增加 `context/*` 不直连 provider 官方 SDK 的边界测试/断言。

## 8. Diagnostics and RuntimeRecorder Additive Fields

- [x] 8.1 在 `runtime/diagnostics` 增加 A67-CTX additive 字段：`context_ref_discover_count`、`context_ref_resolve_count`、`context_edit_estimated_saved_tokens`、`context_edit_gate_decision`、`context_swapback_relevance_score`、`context_lifecycle_tier_stats`、`context_recap_source`。
- [x] 8.2 在 `observability/event.RuntimeRecorder` 接入 A67-CTX 字段映射并保持单写幂等。
- [x] 8.3 增加 QueryRuns parser compatibility 测试（additive + nullable + default）。

## 9. Replay Fixtures and Drift Taxonomy

- [x] 9.1 在 `tool/diagnosticsreplay` 新增 fixtures：`context_reference_first.v1`、`context_isolate_handoff.v1`、`context_edit_gate.v1`、`context_relevance_swapback.v1`、`context_lifecycle_tiering.v1`。
- [x] 9.2 实现 drift 分类：`reference_resolution_drift`、`isolate_handoff_drift`、`edit_gate_threshold_drift`、`swapback_relevance_drift`、`lifecycle_tiering_drift`、`recap_semantic_drift`。
- [x] 9.3 增加 mixed-fixture 回放兼容测试（历史 fixtures + A67-CTX fixtures）。

## 10. Gate and CI Wiring

- [x] 10.1 新增 `scripts/check-context-jit-organization-contract.sh/.ps1`。
- [x] 10.2 将 A67-CTX gate 接入 `scripts/check-quality-gate.sh/.ps1`，保持 shell/PowerShell fail-fast 语义等价。
- [x] 10.3 在 gate 中实现 impacted-contract suites 校验（按 A67-CTX 改动面选择主干 suites）。
- [x] 10.4 在 gate 中实现 `context_provider_sdk_absent` 边界断言（阻断 `context/*` 直连 provider 官方 SDK）。
- [x] 10.5 在 CI 暴露独立 required-check 候选（`context-jit-organization-contract-gate`）。

## 11. Documentation Sync and Validation

- [x] 11.1 更新 `docs/runtime-config-diagnostics.md`（A67-CTX 配置字段、默认值、失败语义、诊断字段）。
- [x] 11.2 更新 `docs/mainline-contract-test-index.md`（A67-CTX fixtures + gate 映射）。
- [x] 11.3 更新 `docs/development-roadmap.md`（A67-CTX 状态与验收口径）。
- [x] 11.4 更新 `README.md`（里程碑快照与能力状态对齐）。
- [x] 11.5 执行最小验证：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`。
- [x] 11.6 执行合同门禁：`check-context-jit-organization-contract.*`、`check-quality-gate.*`、`check-docs-consistency.*`。
- [x] 11.7 记录未执行项与风险说明，确保提案可审查、可回滚、可归档。

验证与风险记录（11.7）：
- `pwsh -File scripts/check-context-jit-organization-contract.ps1` 通过；`bash scripts/check-context-jit-organization-contract.sh` 受当前 Windows/MSYS 执行权限限制（`Bash/Service/CreateInstance/E_ACCESSDENIED` 或 signal pipe error）未能在本环境验证。
- `pwsh -File scripts/check-docs-consistency.ps1` 通过。
- `pwsh -File scripts/check-quality-gate.ps1` 已执行但在本环境出现文件系统权限噪声（临时目录 `rename/unlink Access is denied`），未跑完全部步骤。
- `go test ./...` 在本环境大范围出现 `testing.TempDir RemoveAll cleanup: Access is denied`（多包），属于文件系统权限噪声，非本变更定向失败。
- `go test -race ./...` 跑到 `core/runner::TestSecurityPolicyContractRateLimitDeny` 失败（`tool invoke count = 0, want 1`），其余包多数通过；需在无权限噪声环境复核是否稳定复现。
- `golangci-lint run --config .golangci.yml` 当前报 3 项：`context/assembler/ca3.go` 的 `ineffassign` 及 `runtime/config/runtime_context_jit.go` 两处 `unused`，为现存代码问题，未在本次改动中新引入。
