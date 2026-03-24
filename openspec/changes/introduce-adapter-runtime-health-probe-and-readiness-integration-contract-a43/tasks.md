## 1. Adapter Health Contract Foundation

- [ ] 1.1 在 `adapter` 域新增 runtime health probe 接口与标准结果模型（status/code/message/metadata/checked_at）。
- [ ] 1.2 实现最小默认 probe 执行路径，支持 `probe_timeout` 与 `cache_ttl` 语义。
- [ ] 1.3 固化 adapter-health canonical reason taxonomy，并补齐单测覆盖 unknown/timeout/unavailable/degraded 分支。

## 2. Runtime Config and Readiness Integration

- [ ] 2.1 在 `runtime/config` 增加 `adapter.health.*` 配置域（enabled/strict/probe_timeout/cache_ttl）及默认值。
- [ ] 2.2 补齐启动校验与热更新 fail-fast + 原子回滚测试（非法 duration、非法布尔覆盖等）。
- [ ] 2.3 在 readiness preflight 接入 adapter-health 映射逻辑，覆盖 strict/non-strict 下 required/optional 判定路径。
- [ ] 2.4 保持 readiness finding schema 与 canonical code 稳定，补齐重复 preflight 的 determinism 测试。

## 3. Diagnostics and Query Surface

- [ ] 3.1 在 `runtime/diagnostics` 增加 adapter-health additive 字段与聚合计数（status/probe_total/degraded_total/unavailable_total/primary_code）。
- [ ] 3.2 确保 QueryRuns 与相关输出保持 `additive + nullable + default` 兼容窗口。
- [ ] 3.3 补齐 replay idempotency 测试，验证 adapter-health 聚合不膨胀。

## 4. Conformance and Quality Gates

- [ ] 4.1 在 `integration/adapterconformance` 增加 adapter-health 矩阵（required unavailable、optional unavailable downgrade、degraded visibility）。
- [ ] 4.2 确保 adapter-health conformance 套件离线 deterministic，可在无网络环境运行。
- [ ] 4.3 更新 `scripts/check-adapter-conformance.*` 与 `scripts/check-quality-gate.*`，纳入 adapter-health 阻断步骤并保持 shell/PowerShell parity。

## 5. Documentation and Acceptance

- [ ] 5.1 更新 `docs/runtime-config-diagnostics.md`，补充 `adapter.health.*` 字段、默认值、校验与诊断字段说明。
- [ ] 5.2 更新 `docs/mainline-contract-test-index.md` 与 `docs/development-roadmap.md`，补充 A43 映射与状态。
- [ ] 5.3 更新 `README.md` 与 `adapter/README.md`，增加 adapter runtime health 能力描述与边界说明。
- [ ] 5.4 执行并记录最小验收命令：`go test ./...`、`go test -race ./...`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-quality-gate.ps1`。
