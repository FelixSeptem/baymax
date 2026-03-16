## 1. Config and Rule Reuse

- [x] 1.1 保持 `drop_low_priority` 配置模型不变（`priority_by_tool`/`priority_by_keyword`/`droppable_priorities`），复用到 mcp/skill 路径
- [x] 1.2 校验默认策略不变：未显式配置时仍为 `backpressure=block`
- [x] 1.3 补齐配置与校验测试，确保扩展路径不引入新枚举或新字段

## 2. Dispatcher and Runner Semantics

- [x] 2.1 将 `drop_low_priority` 背压逻辑扩展到 mcp 调度路径
- [x] 2.2 将 `drop_low_priority` 背压逻辑扩展到 skill 调度路径
- [x] 2.3 保持 local/mcp/skill 三路径 drop 判定语义一致
- [x] 2.4 在 runner 收敛“当轮全量 drop 立即 fail-fast”到三路径
- [x] 2.5 保持 Run/Stream 在 drop 命中与全量 drop 场景下语义一致（错误分类、终止条件）

## 3. Timeline and Diagnostics

- [x] 3.1 在 mcp/skill 路径发射统一 timeline reason：`backpressure.drop_low_priority`
- [x] 3.2 为 diagnostics 增加按来源分桶计数（`local/mcp/skill`）
- [x] 3.3 保持既有 aggregate 字段兼容并验证分桶汇总关系

## 4. Contract Tests and Benchmarks

- [x] 4.1 新增/更新契约测试：local/mcp/skill 在 Run/Stream 的 drop 语义一致
- [x] 4.2 新增/更新契约测试：三路径“当轮全量 drop”均 fail-fast
- [x] 4.3 新增 benchmark：mcp/skill drop_low_priority 场景，并输出 `p95-ns/op`
- [x] 4.4 更新 `docs/mainline-contract-test-index.md` 登记新增/变更用例

## 5. Documentation Sync

- [x] 5.1 更新 `README.md`（drop_low_priority 适用域扩展为 local+mcp+skill，默认策略不变）
- [x] 5.2 更新 `docs/runtime-config-diagnostics.md`（统一规则语义 + diagnostics 分桶字段）
- [x] 5.3 更新 `docs/v1-acceptance.md`（三路径一致性与 fail-fast 验收条目）
- [x] 5.4 更新 `docs/development-roadmap.md`（标记 R7 落点）

## 6. Validation

- [x] 6.1 执行 `go test ./...`
- [x] 6.2 执行 `go test -race ./...`
- [x] 6.3 执行 `golangci-lint run --config .golangci.yml`
