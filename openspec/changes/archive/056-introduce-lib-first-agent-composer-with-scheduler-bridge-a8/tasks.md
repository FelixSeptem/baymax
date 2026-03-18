## 1. Composer 包与接缝 API

- [x] 1.1 新增 `orchestration/composer` 包与最小公开 API（初始化、Run、Stream、依赖注入）。
- [x] 1.2 在 composer 中接入 `runner/workflow/teams/a2a/scheduler` 组合装配，移除宿主手工拼装必需路径。
- [x] 1.3 为 composer 提供最小示例构建器或 options，保证 `library-first` 接入可读性。

## 2. Scheduler Bridge 与双子任务路径

- [x] 2.1 在 composer 中实现 scheduler-managed 子任务桥接，支持 `local child-run` 路由。
- [x] 2.2 在 composer 中实现 scheduler-managed 子任务桥接，支持 `a2a child-run` 路由。
- [x] 2.3 统一 local/a2a 子任务 terminal commit 收口与幂等键语义（避免重复回放放大计数）。

## 3. 配置生效与降级策略

- [x] 3.1 实现 scheduler backend 初始化失败自动降级到 `memory` 的逻辑与原因码输出。
- [x] 3.2 实现 scheduler/subagent 热更新 `next_attempt_only` 生效边界（in-flight attempt 不回溯变更）。
- [x] 3.3 固化 spawn 前 guardrail fail-fast（`max_depth/max_active_children/child_timeout_budget`）并发射 `subagent.budget_reject`。

## 4. 观测、摘要与边界约束

- [x] 4.1 在 composer 路径补齐 `run.finished` additive 摘要字段注入，保持 `RuntimeRecorder` 单写入口。
- [x] 4.2 保证 composer 管理路径的 timeline reason 继续使用既有 canonical 命名空间。
- [x] 4.3 增加边界断言，禁止 composer/scheduler 直接写入 `runtime/diagnostics`。

## 5. 契约测试与质量门禁

- [x] 5.1 新增 composer integration contract tests：Run/Stream 语义一致（终态类别 + 聚合字段）。
- [x] 5.2 新增 composer integration contract tests：scheduler fallback-to-memory、takeover 与 replay-idempotency。
- [x] 5.3 将 composer suite 并入 `scripts/check-multi-agent-shared-contract.sh/.ps1` 现有阻断路径。
- [x] 5.4 更新 `tool/contributioncheck` 快照校验，覆盖 composer 新增契约条目。

## 6. 文档与示例对齐

- [x] 6.1 更新 `README.md`，新增 composer 主路径接入说明与配置示例。
- [x] 6.2 更新 `docs/runtime-config-diagnostics.md`、`docs/runtime-module-boundaries.md`、`docs/v1-acceptance.md` 的 A8 条款。
- [x] 6.3 更新 `docs/mainline-contract-test-index.md` 与 `docs/development-roadmap.md` 的 A8 映射行。
- [x] 6.4 升级 `examples/07` 与 `examples/08` 至 composer 接入范式并补对应 README 说明。

## 7. 回归验证

- [x] 7.1 执行 `go test ./...` 并记录结果。
- [x] 7.2 执行 `$env:CGO_ENABLED='1'; go test -race ./...` 并记录结果。
- [x] 7.3 执行 `golangci-lint run --config .golangci.yml` 并记录结果。
- [x] 7.4 执行 `pwsh -File scripts/check-multi-agent-shared-contract.ps1` 与 `pwsh -File scripts/check-docs-consistency.ps1` 并记录结果。

