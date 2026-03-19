## 1. Delayed Dispatch Core Model

- [x] 1.1 在 `scheduler.Task` 增加可选 `not_before` 字段并完成标准化校验。
- [x] 1.2 在 memory/file store 序列化与恢复链路中持久化 `not_before`。
- [x] 1.3 明确 `not_before` 时间语义（空值、过去时间、未来时间）并补单测。

## 2. Claim Eligibility and Scheduler Integration

- [x] 2.1 更新 claim 可领取判定，加入 `not_before` gate。
- [x] 2.2 组合 delayed gate 与 `next_eligible_at` retry gate。
- [x] 2.3 保证 delayed 任务在到期后仍按现有 QoS/fairness 规则选取。
- [x] 2.4 补 scheduler 单测覆盖 ready/non-ready 混合队列行为。

## 3. Composer and Multi-Agent Path

- [x] 3.1 在 composer child dispatch 请求中支持 `not_before` 透传。
- [x] 3.2 补 composer delayed child 路径回归测试（含 Run/Stream 等价）。
- [x] 3.3 验证 delayed dispatch 与 A12 异步回报协作语义不冲突。

## 4. Observability and Diagnostics

- [x] 4.1 增加 delayed-dispatch timeline reason 与关联字段映射。
- [x] 4.2 增加 run diagnostics delayed additive 字段（total/claim/wait p95）。
- [x] 4.3 保证 delayed 字段遵循 `additive + nullable + default`。

## 5. Contract Tests and Gates

- [x] 5.1 新增 integration 合同测试：提前不可领取、到期可领取。
- [x] 5.2 新增恢复合同测试：重启恢复后不提前执行。
- [x] 5.3 新增 delayed 场景 Run/Stream 语义等价测试。
- [x] 5.4 将 delayed suites 并入 `check-multi-agent-shared-contract.sh/.ps1`。
- [x] 5.5 更新 `tool/contributioncheck` 快照契约映射。

## 6. Docs and Validation

- [x] 6.1 更新 `README.md` delayed dispatch 最小示例。
- [x] 6.2 更新 `docs/runtime-config-diagnostics.md` delayed 语义与字段说明。
- [x] 6.3 更新 `docs/mainline-contract-test-index.md` A13 测试映射。
- [x] 6.4 更新 `docs/development-roadmap.md` A13 状态条目。
- [x] 6.5 执行 `go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-multi-agent-shared-contract.ps1`。
