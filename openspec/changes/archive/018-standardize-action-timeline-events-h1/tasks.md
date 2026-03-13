## 1. Timeline Contract And Types

- [x] 1.1 在 `core/types` 定义 Action Timeline 事件 DTO、phase 枚举（含 `context_assembler`）与状态枚举（`pending|running|succeeded|failed|skipped|canceled`）
- [x] 1.2 为 timeline 字段补充最小兼容约束，确保新增结构化事件不破坏既有事件消费路径

## 2. Runner/Event Integration

- [x] 2.1 在 `core/runner` 与相关事件发布路径接入 timeline 发射，覆盖 Run/Stream 双路径
- [x] 2.2 在 `observability/event` 统一 timeline 事件写入与序列顺序语义（phase transition deterministic）
- [x] 2.3 将 `context_assembler` 作为独立 phase 输出并校验在未触发时不产生伪事件

## 3. Tests And Quality Gates

- [x] 3.1 新增/更新契约测试：Run/Stream 成功、失败、跳过、取消路径的 timeline 语义一致性
- [x] 3.2 新增/更新兼容性测试：既有事件字段消费不受 timeline 引入影响
- [x] 3.3 执行质量门禁：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`

## 4. Documentation And TODO Traceability

- [x] 4.1 更新 `README.md`：补充 Action Timeline 事件、phase/status 语义与默认启用说明
- [x] 4.2 更新 `docs/runtime-config-diagnostics.md`：声明 H1 不新增 diagnostics 聚合字段，并增加 timeline observability 收敛 TODO
- [x] 4.3 更新 `docs/development-roadmap.md`：补充 H1 完成后到聚合可观测性的后续节奏与里程碑 TODO
- [x] 4.4 执行 docs 一致性检查并修复差异（`scripts/check-docs-consistency.ps1`）
