## 1. CA2 Contracts And Configuration

- [x] 1.1 在 `core/types` 扩展 CA2 stage 装配请求/响应结构（含 stage 状态、skip reason、recap 状态）
- [x] 1.2 在 `runtime/config` 增加 CA2 配置字段（stage 开关、routing mode、stage 失败策略、timeout、provider、tail recap）
- [x] 1.3 增加 CA2 配置校验（枚举/阈值/占位 provider fail-fast）并保持 `env > file > default`

## 2. Stage Routing And Provider Layer

- [x] 2.1 在 `context/assembler` 实现 Stage1 -> Stage2 的规则化路由执行框架
- [x] 2.2 新增 Stage2 provider 接口层并实现 file provider（rag/db 返回 not-ready）
- [x] 2.3 预留 agentic routing 扩展 hook（TODO），默认仅启用规则路由
- [x] 2.4 实现 tail recap 末尾追加（最小字段 `status/decisions/todo/risks`）与长度/脱敏处理

## 3. Diagnostics And Runner Integration

- [x] 3.1 在 `runtime/diagnostics` 扩展 CA2 枚举与字段（stage status、skip reason、stage latency、recap status）
- [x] 3.2 将 CA2 结果通过现有 `run.finished -> RuntimeRecorder -> diagnostics.Store` 管道写入
- [x] 3.3 在 `core/runner` 接入 CA2 结果并确保 Run/Stream 与 complete-only 语义不回退

## 4. Tests And Quality Gates

- [x] 4.1 新增 CA2 单元测试（routing 命中/跳过、stage 策略、file provider、placeholder provider fail-fast、tail recap schema）
- [x] 4.2 新增 runner 集成回归（Run/Stream 语义一致、事件契约不变、stage 降级可观测）
- [x] 4.3 执行并通过 `go test ./...`、`go test -race ./...` 与 `golangci-lint run --config .golangci.yml`

## 5. Documentation Alignment

- [x] 5.1 更新 README 的 CA2 状态、配置示例与边界说明
- [x] 5.2 更新 `docs/runtime-config-diagnostics.md` 的 CA2 配置与诊断字段
- [x] 5.3 更新 `docs/development-roadmap.md`、`docs/context-assembler-phased-plan.md`、`docs/v1-acceptance.md`（标注 CA2 进展与 examples TODO）
