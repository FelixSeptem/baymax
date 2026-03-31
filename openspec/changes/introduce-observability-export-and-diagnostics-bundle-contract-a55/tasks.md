## 1. Runtime Config Schema and Validation

- [ ] 1.1 在 `runtime/config` 新增 `runtime.observability.export.*` 与 `runtime.diagnostics.bundle.*` 字段、默认值与 schema 解析。
- [ ] 1.2 实现配置优先级 `env > file > default`，并补齐 enum/path/profile 组合校验。
- [ ] 1.3 实现热更新原子切换与失败回滚（invalid profile、invalid output dir、invalid policy）。
- [ ] 1.4 补齐配置单测（启动 fail-fast、热更新回滚、优先级覆盖场景）。

## 2. Exporter SPI and Profile Resolver

- [ ] 2.1 新增 observability exporter SPI（profile resolver + canonical error taxonomy）。
- [ ] 2.2 落地 `none|otlp|langfuse|custom` profile 基线与参数归一化逻辑。
- [ ] 2.3 将 exporter 消费路径接入 `RuntimeRecorder` 后置链路，禁止绕过 single-writer 直接导出。
- [ ] 2.4 实现 exporter 队列与失败策略（`fail_fast|degrade_and_record`、overflow 行为）并补齐并发测试。

## 3. Diagnostics Bundle Contract Implementation

- [ ] 3.1 定义并实现 bundle manifest schema（`schema_version`、metadata、section 清单）。
- [ ] 3.2 实现 bundle 生成器，输出 timeline/diagnostics/redacted config/replay hints/gate fingerprint。
- [ ] 3.3 复用 runtime redaction 策略，确保 bundle 不持久化敏感明文。
- [ ] 3.4 补齐 bundle 生成与失败路径测试（不可写目录、超大小阈值、section 缺失）。

## 4. Diagnostics and RuntimeRecorder Field Integration

- [ ] 4.1 在 `runtime/diagnostics` 增加 export/bundle additive 字段并保持 `additive + nullable + default`。
- [ ] 4.2 在 `observability/event.RuntimeRecorder` 增加 export/bundle 事件映射与计数聚合。
- [ ] 4.3 校验字段 bounded-cardinality 与 replay idempotency 约束。
- [ ] 4.4 补齐 Run/Stream export/bundle 语义等价测试。

## 5. Readiness and Admission Alignment

- [ ] 5.1 在 `runtime/config/readiness` 增加 `observability.export.*` 与 `diagnostics.bundle.*` finding 分类。
- [ ] 5.2 固化 strict/non-strict 映射语义并补齐 deterministic primary finding 断言。
- [ ] 5.3 补齐 readiness 相关集成测试（sink unavailable、profile invalid、bundle output unavailable）。

## 6. Replay Fixture and Drift Classification

- [ ] 6.1 在 `tool/diagnosticsreplay` 新增 `observability.v1` fixture schema、loader 与 normalization。
- [ ] 6.2 新增 drift 分类断言（profile/status/reason/schema/redaction/fingerprint）。
- [ ] 6.3 增加 mixed fixture 回放兼容测试（A52/A53/A54 + `observability.v1`）。

## 7. Quality Gate and CI Wiring

- [ ] 7.1 新增 `scripts/check-observability-export-and-bundle-contract.sh/.ps1` 并接入 `check-quality-gate.*`。
- [ ] 7.2 在 CI 暴露独立 required-check 候选（observability export + bundle gate）。
- [ ] 7.3 校验 shell/PowerShell gate parity（失败传播、退出码与阻断语义一致）。

## 8. Docs and Roadmap Sync

- [ ] 8.1 更新 `docs/runtime-config-diagnostics.md`（新配置域、additive 字段、reason taxonomy）。
- [ ] 8.2 更新 `docs/mainline-contract-test-index.md`（A55 replay/gate 索引）。
- [ ] 8.3 更新 `docs/development-roadmap.md` 与 `README.md` 状态快照（A55 proposal active）。
- [ ] 8.4 执行并记录验收命令：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-quality-gate.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`。
