## 1. Scheduler QoS 配置与数据模型

- [x] 1.1 在 `runtime/config` 增加 `scheduler.qos.*` 配置域并保持默认 `mode=fifo`。
- [x] 1.2 增加 `fairness.max_consecutive_claims_per_priority`，默认值设置为 `3` 并补校验测试。
- [x] 1.3 增加 `scheduler.dlq.*` 配置域并保持默认 `enabled=false`。
- [x] 1.4 增加 `scheduler.retry.backoff.*` 配置域，支持指数退避 + 抖动参数校验。

## 2. Scheduler 领取与重试治理实现

- [x] 2.1 在 scheduler claim 路径实现可选 priority 模式（优先级来源为 task 字段）。
- [x] 2.2 实现 fairness 窗口逻辑（同优先级连续 claim 上限 3 后让渡）。
- [x] 2.3 实现指数退避 + 抖动的重试调度逻辑。
- [x] 2.4 实现 retry 超限后的 dead-letter 转移与常规重试终止。

## 3. 观测与契约字段收敛

- [x] 3.1 增加 qos/fairness/backoff/dlq timeline reason 与 payload 字段映射。
- [x] 3.2 增加 run 摘要 additive 统计字段并通过 `RuntimeRecorder` 入库。
- [x] 3.3 保证新增字段遵循 `additive + nullable + default` 兼容窗口。

## 4. 契约测试与门禁

- [x] 4.1 新增 integration 套件：priority 顺序、公平窗口、防饥饿回归。
- [x] 4.2 新增 integration 套件：指数退避抖动、DLQ 转移、重放幂等。
- [x] 4.3 补 Run/Stream 等价测试（scheduler-managed QoS 路径）。
- [x] 4.4 将 suite 并入 `scripts/check-multi-agent-shared-contract.sh/.ps1` 阻断路径。
- [x] 4.5 扩展 `tool/contributioncheck` 快照校验（reason/字段/门禁条目）。

## 5. 文档与索引同步

- [x] 5.1 更新 `docs/runtime-config-diagnostics.md` 的 qos/dlq/backoff 配置与字段说明。
- [x] 5.2 更新 `docs/runtime-module-boundaries.md` 的 scheduler qos 边界约束。
- [x] 5.3 更新 `docs/mainline-contract-test-index.md` 添加 A10 测试映射。
- [x] 5.4 更新 `docs/v1-acceptance.md` 与 `docs/development-roadmap.md` 的状态条目。
- [x] 5.5 更新 `README.md` 添加 scheduler QoS 与 DLQ 最小配置示例。

## 6. 回归验证

- [x] 6.1 执行 `go test ./...`。
- [x] 6.2 执行 `$env:CGO_ENABLED='1'; go test -race ./...`。
- [x] 6.3 执行 `golangci-lint run --config .golangci.yml`。
- [x] 6.4 执行 `pwsh -File scripts/check-multi-agent-shared-contract.ps1` 与 `pwsh -File scripts/check-docs-consistency.ps1`。
