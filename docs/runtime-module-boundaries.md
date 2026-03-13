# Runtime Module Boundaries

更新时间：2026-03-13

## 目标

明确全局 runtime 平台能力与 MCP 子域能力边界，避免配置与诊断入口继续耦合在单个 MCP runtime 包。

## 模块职责

- `runtime/config`
  - 统一配置加载（YAML + env + default）
  - 配置校验与 fail-fast 启动
  - 热更新与原子快照切换
  - MCP profile 解析（作为配置字段的一部分）
- `runtime/diagnostics`
  - 统一诊断数据模型与有界存储
  - `call/run/reload/skill` 记录与查询
  - 配置脱敏输出辅助
- `mcp/profile`
  - MCP profile 常量与策略解析（仅 MCP 语义）
- `mcp/retry`
  - MCP 重试控制（retryable 分类 + backoff）
- `mcp/diag`
  - MCP 调用摘要字段模型与本地有界缓存
- `mcp/internal/reliability`
  - MCP 内部共享重试/超时/backoff 执行骨架（internal-only）
- `mcp/internal/observability`
  - MCP 内部共享事件发射与诊断映射桥接（internal-only）
- `mcp/http` / `mcp/stdio`
  - 传输实现
  - 消费 `runtime/config.Manager` 配置与诊断 API
- `core/runner` / `tool/local` / `skill/loader`
  - 消费全局 runtime 配置快照
  - 产出标准运行时事件（不直接写诊断存储）
- `observability/event`
  - 事件日志与分发
  - `RuntimeRecorder` 作为诊断唯一写入入口，将事件映射为统一诊断记录

## 依赖方向

允许方向（简化）：

`runtime/*` -> (no dependency on `mcp/http` or `mcp/stdio`)

`mcp/*`, `core/*`, `tool/*`, `skill/*`, `observability/*` -> `runtime/*`

禁止方向：

- `runtime/config` 或 `runtime/diagnostics` 反向依赖 `mcp/http` / `mcp/stdio`
- 非 `mcp/*` 包依赖 `mcp/internal/*`

CI 通过 `scripts/check-runtime-boundaries.sh` 做静态检查。
治理型评审可结合 `docs/modular-e2e-review-matrix.md` 执行“模块 + 主干链路”双视角核验。

## Owner 建议

- `runtime/config`：平台基础设施 owner
- `runtime/diagnostics`：可观测性 owner
- `mcp/profile`、`mcp/retry`、`mcp/diag`：MCP owner
- `skill/loader`：Skill owner

## 扩展约束

- 新增全局配置字段时，必须同步：
  - `runtime/config` schema + validation
  - `docs/runtime-config-diagnostics.md` 字段索引
- 新增诊断记录类型时，必须同步：
  - `runtime/diagnostics` record 定义
  - 文档中的字段与语义说明

## 全局限制（职责分工重点）

- Context Assembler 与 Model Provider 的职责必须分离：
  - `context/assembler` 只做策略编排与触发时机控制（例如 CA3 压力分区、阈值判定、计数调用节流）。
  - `model/*` 负责 provider 协议细节与官方 SDK 调用（包括 token count、能力探测、流式映射）。
- 禁止在 `context/*` 中直接引入 provider 官方 SDK（OpenAI/Anthropic/Gemini），避免跨层耦合与升级扩散。
- 任何新增 provider 级能力（例如 token count、模型元数据查询）应先落在 `model/<provider>`，再由上层通过接口复用。
