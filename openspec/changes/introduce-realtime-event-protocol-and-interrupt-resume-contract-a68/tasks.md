## 1. Runtime Realtime Config Schema and Validation

- [ ] 1.1 在 `runtime/config` 增加 `runtime.realtime.protocol.*` 字段、默认值与 env 映射。
- [ ] 1.2 在 `runtime/config` 增加 `runtime.realtime.interrupt_resume.*` 字段、默认值与 env 映射。
- [ ] 1.3 实现配置校验（枚举、边界、组合约束）并保持 fail-fast。
- [ ] 1.4 增加配置优先级测试（`env > file > default`）。
- [ ] 1.5 增加热更新非法配置回滚测试（原子回滚）。

## 2. Realtime Event Envelope and Taxonomy

- [ ] 2.1 定义 canonical event envelope（`event_id/session_id/run_id/seq/type/ts/payload`）。
- [ ] 2.2 定义 canonical event taxonomy（request/delta/interrupt/resume/ack/error/complete）。
- [ ] 2.3 实现 envelope 序列化/反序列化与结构校验。
- [ ] 2.4 增加事件 schema 单测（缺字段、错类型、非法 seq）。

## 3. Sequence, Dedup, and Idempotency Semantics

- [ ] 3.1 实现单调 `seq` 推进与 gap 检测。
- [ ] 3.2 实现 dedup key 与重复事件幂等吸收语义。
- [ ] 3.3 增加重复 interrupt/resume 输入不膨胀计数测试。
- [ ] 3.4 增加 sequence gap 和乱序输入分类测试。

## 4. Interrupt/Resume Runtime Wiring

- [ ] 4.1 在 `core/runner` 接入 interrupt 状态冻结与 resume cursor 记录。
- [ ] 4.2 接入 resume 恢复路径（合法游标恢复、非法游标拒绝）。
- [ ] 4.3 增加 interrupt/resume 状态机转换测试。
- [ ] 4.4 保证 A56 终止 taxonomy 不变，不引入平行终止通道。

## 5. Run/Stream Parity and Boundaries

- [ ] 5.1 增加 equivalent Run/Stream interrupt/resume parity 集成测试。
- [ ] 5.2 增加 realtime 错误分层（transport/protocol/semantic）映射一致性测试。
- [ ] 5.3 增加 A58/A67 解释字段复用与边界回归测试（不引入第二套解释链）。

## 6. Diagnostics and RuntimeRecorder Additive Fields

- [ ] 6.1 在 `runtime/diagnostics` 增加 A68 additive 字段：`realtime_protocol_version`、`realtime_event_seq_max`、`realtime_interrupt_total`、`realtime_resume_total`、`realtime_resume_source`、`realtime_idempotency_dedup_total`、`realtime_last_error_code`。
- [ ] 6.2 在 `observability/event.RuntimeRecorder` 接入 A68 字段映射并保持单写幂等。
- [ ] 6.3 增加 QueryRuns parser compatibility 测试（additive + nullable + default）。

## 7. Replay Fixture and Drift Taxonomy

- [ ] 7.1 在 `tool/diagnosticsreplay` 新增 `realtime_event_protocol.v1` fixture schema 与 loader。
- [ ] 7.2 实现 drift 分类：`realtime_event_order_drift`、`realtime_interrupt_semantic_drift`、`realtime_resume_semantic_drift`、`realtime_idempotency_drift`、`realtime_sequence_gap_drift`。
- [ ] 7.3 增加 mixed-fixture 回放兼容测试（历史 fixtures + A68 fixture）。

## 8. Gate and CI Wiring

- [ ] 8.1 新增 `scripts/check-realtime-protocol-contract.sh/.ps1`。
- [ ] 8.2 将 A68 gate 接入 `scripts/check-quality-gate.sh/.ps1`，保持 shell/PowerShell fail-fast 语义等价。
- [ ] 8.3 在 gate 中实现 `realtime_control_plane_absent` 断言（禁止托管实时网关/控制面依赖）。
- [ ] 8.4 在 gate 中实现 impacted-contract suites 校验（按 A68 影响面选择主干 suites）。
- [ ] 8.5 在 CI 暴露独立 required-check 候选（`realtime-protocol-contract-gate`）。

## 9. Documentation Sync

- [ ] 9.1 更新 `docs/runtime-config-diagnostics.md`（A68 配置字段、默认值、失败语义、诊断字段）。
- [ ] 9.2 更新 `docs/mainline-contract-test-index.md`（A68 fixture + gate 映射）。
- [ ] 9.3 更新 `docs/development-roadmap.md`（A68 状态与验收口径）。
- [ ] 9.4 更新 `README.md`（里程碑快照与能力状态对齐）。

## 10. Validation and Exit

- [ ] 10.1 执行最小验证：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`。
- [ ] 10.2 执行合同门禁：`check-realtime-protocol-contract.*`、`check-quality-gate.*`、`check-docs-consistency.*`。
- [ ] 10.3 记录未执行项与风险说明，确保提案可审查、可回滚、可归档。
