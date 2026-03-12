## 1. CA1 Config And Contracts

- [x] 1.1 在 `core/types` 增加 context assembler CA1 请求/响应契约与 prefix hash 元数据结构
- [x] 1.2 在 `runtime/config` 增加 `context_assembler` 最小配置字段与默认值（enabled 默认 true，guard.fail_fast 默认 true）
- [x] 1.3 为 context storage 增加 backend 接口与 `file`/`db` 选择逻辑（db 仅占位并返回明确 unsupported）

## 2. Assembler Baseline Implementation

- [x] 2.1 新增 `context/assembler` 骨架并接入 `core/runner` pre-model hook（Run/Stream 双路径）
- [x] 2.2 实现 immutable prefix 构建与 `prefix_hash` 校验逻辑（同 session/version 漂移即 fail-fast）
- [x] 2.3 实现 `context/journal` 本地 JSONL append-only 写入（intent/commit）且禁止中间插入/重排
- [x] 2.4 实现 `context/guard` 基础规则防护（hash/schema/sanitize）并保持规则独立于 LLM

## 3. Diagnostics And Observability

- [x] 3.1 在 `runtime/diagnostics` 扩展 assembler CA1 最小字段：`prefix_hash`、`assemble_latency_ms`、`assemble_status`、`guard_violation`
- [x] 3.2 将 assembler 结果通过现有 event/diagnostics 管道写入，保持 single-writer 与 idempotency 语义

## 4. Tests And Quality Gates

- [x] 4.1 新增 CA1 单元测试（prefix 一致性、append-only、guard fail-fast、db placeholder 拒绝）
- [x] 4.2 新增 runner 集成回归（Run/Stream 语义兼容，不破坏 complete-only 事件契约）
- [x] 4.3 执行并通过 `go test ./...`、`go test -race ./...` 与 `golangci-lint run --config .golangci.yml`

## 5. Documentation Alignment

- [x] 5.1 更新 README 的 Context Assembler CA1 状态与启用默认值说明
- [x] 5.2 更新 `docs/runtime-config-diagnostics.md` 的 CA1 配置/诊断字段说明
- [x] 5.3 更新 `docs/development-roadmap.md`、`docs/context-assembler-phased-plan.md`、`docs/v1-acceptance.md` 以反映 CA1 完成与 CA2+ 边界
