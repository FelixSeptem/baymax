## 1. CA2 Agentic Router Core

- [x] 1.1 在 `context/assembler` 定义 agentic router callback 接口与决策结果结构（含 `run_stage2/reason`）。
- [x] 1.2 在 assembler 构造与 option 注入链路接入 callback 注册能力（保持未注册时可回退）。
- [x] 1.3 改造 `applyCA2`：`routing_mode=agentic` 时执行 callback 决策，成功路径按决策触发/跳过 Stage2。
- [x] 1.4 实现 callback 失败归一化处理（missing/timeout/error/invalid），统一按 `best_effort` 回退 `rules`。

## 2. Runtime Config Integration

- [x] 2.1 在 `runtime/config` 增加 `context_assembler.ca2.agentic.*` 配置结构与默认值（超时、失败策略）。
- [x] 2.2 增加配置加载与环境变量映射，保持 `env > file > default` 语义。
- [x] 2.3 增加 startup/hot-reload 校验：非法超时与非法失败策略 fail-fast，热更新失败回滚。

## 3. Diagnostics And Event Mapping

- [x] 3.1 在 CA2 结果/诊断模型中新增路由字段：`stage2_router_mode`、`stage2_router_decision`、`stage2_router_reason`、`stage2_router_latency_ms`、`stage2_router_error`。
- [x] 3.2 打通 `core/runner` -> `observability/event` -> `runtime/diagnostics` 的字段映射与持久化。
- [x] 3.3 为 fallback 场景补齐标准化 reason/error 映射，确保与现有 stage policy 字段并存且不冲突。

## 4. Contract Tests And Regression

- [x] 4.1 新增/更新 assembler 单测覆盖 agentic callback 成功决策（run_stage2=true/false）路径。
- [x] 4.2 新增/更新 assembler 单测覆盖 callback 失败回退（missing/timeout/error/invalid）路径。
- [x] 4.3 新增/更新 runner 契约测试，验证 Run/Stream 在 agentic 决策与 fallback 场景的语义等价。
- [x] 4.4 执行并通过回归门禁：`go test ./...`、`go test -race ./...` 与相关 CA2/diagnostics 契约测试。

## 5. Docs Alignment

- [x] 5.1 更新 `docs/runtime-config-diagnostics.md` 的 CA2 agentic 配置示例、默认值与诊断字段说明。
- [x] 5.2 更新 `docs/v1-acceptance.md`，移除 agentic not-ready 描述并补齐新能力与限制说明。
- [x] 5.3 更新 `docs/development-roadmap.md` 的相关进展条目，确保与提案能力口径一致。
