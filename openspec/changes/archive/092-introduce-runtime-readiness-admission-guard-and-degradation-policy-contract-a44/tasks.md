## 1. Runtime Admission Config and Policy Core

- [x] 1.1 在 `runtime/config` 增加 `runtime.readiness.admission.*` 配置结构与默认值（enabled/mode/block_on/degraded_policy）。
- [x] 1.2 补齐 admission 配置 startup 校验与 hot reload fail-fast + 回滚测试。
- [x] 1.3 在 `runtime/config` 实现 readiness admission 决策器，复用 preflight 结果并输出 canonical 决策分类。

## 2. Composer Run/Stream Admission Guard Wiring

- [x] 2.1 在 `orchestration/composer` managed Run 入口接入 admission guard。
- [x] 2.2 在 `orchestration/composer` managed Stream 入口接入 admission guard，保持与 Run 语义一致。
- [x] 2.3 实现 deny 路径“无副作用”保障并补齐回归测试（不 enqueue、不变更 task lifecycle、不写 mailbox）。

## 3. Diagnostics and Query Compatibility

- [x] 3.1 在 `runtime/diagnostics` 增加 readiness-admission additive 字段与计数聚合。
- [x] 3.2 确保 QueryRuns 输出遵循 `additive + nullable + default` 兼容窗口。
- [x] 3.3 补齐 replay idempotency 测试，验证 admission 聚合计数不膨胀。

## 4. Contract Tests and Quality Gates

- [x] 4.1 新增 integration 合同测试：blocked fail-fast、degraded allow/fail policy、Run/Stream 等价。
- [x] 4.2 新增 deny-path side-effect-free 套件，验证调度/队列状态零副作用。
- [x] 4.3 更新 `scripts/check-quality-gate.*`，将 readiness-admission suites 纳入阻断步骤并保持 shell/PowerShell parity。

## 5. Documentation and Acceptance

- [x] 5.1 更新 `docs/runtime-config-diagnostics.md`，补充 `runtime.readiness.admission.*` 字段与诊断说明。
- [x] 5.2 更新 `docs/mainline-contract-test-index.md` 与 `docs/development-roadmap.md`，补齐 A44 行与状态。
- [x] 5.3 更新 `README.md` 与 `orchestration/README.md` 的 admission 行为说明与默认策略。
- [x] 5.4 执行并记录最小验收命令：`go test ./...`、`go test -race ./...`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-quality-gate.ps1`。
