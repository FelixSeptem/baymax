## 1. Diagnostics Model Extension

- [x] 1.1 扩展 `runtime/diagnostics` run record，新增 timeline phase 聚合字段（`count_total/failed_total/canceled_total/skipped_total/latency_ms/latency_p95_ms`）
- [x] 1.2 设计并实现 phase 聚合数据结构（按 phase 维度）并保持向后兼容
- [x] 1.3 明确 `latency_p95_ms` 计算规则并补充边界处理（空样本/单样本/重复样本）

## 2. Recorder Aggregation And Idempotency

- [x] 2.1 在 `observability/event` 到 `runtime/diagnostics` 路径接入 timeline 聚合写入
- [x] 2.2 实现同 run timeline 重放幂等去重，确保重复事件不重复计数
- [x] 2.3 保持 H1 既有结构化 timeline 事件和非 timeline 事件兼容消费

## 3. Contract Tests And Quality Gates

- [x] 3.1 新增/更新契约测试：Run/Stream 同场景 phase 状态分布与聚合统计等价
- [x] 3.2 新增/更新幂等测试：重复 timeline 重放不改变聚合结果
- [x] 3.3 执行质量门禁：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`

## 4. Docs Alignment

- [x] 4.1 更新 `README.md`：补充 H1.5 timeline 聚合可观测字段说明
- [x] 4.2 更新 `docs/runtime-config-diagnostics.md`：将 H1 TODO 收敛为 H1.5 已落地能力
- [x] 4.3 更新 `docs/development-roadmap.md`：标记 H1.5 进展并同步后续 TODO
- [x] 4.4 执行 docs 一致性检查（`scripts/check-docs-consistency.ps1`）
