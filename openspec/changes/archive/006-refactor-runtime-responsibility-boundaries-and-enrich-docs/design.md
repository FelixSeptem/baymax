## Context

上一阶段已实现运行时配置加载、热更新与诊断 API，但最初实现为 MCP 单体 runtime 包，语义上将“通用运行时能力”与“MCP 子域能力”混合在一起。随着配置与诊断能力计划扩展到 runner/tool/skill/observability，该结构会持续放大跨模块耦合，增加维护和迁移成本。

此次变更是跨模块职责重构：保持语义一致与 fail-fast 基线不变，调整代码结构与依赖方向，并同步补齐架构与迁移文档。

## Goals / Non-Goals

**Goals:**
- 建立清晰的 runtime 模块边界：通用配置中心与 MCP 专属策略解耦。
- 建立清晰的 runtime 模块边界：全局诊断 API 与 MCP 诊断字段模型解耦。
- 将配置快照接口扩展为全局依赖注入入口（runner/tool/skill/observability/MCP）。
- 将诊断 API 扩展为全局依赖注入入口（runner/tool/skill/observability/MCP）。
- 保持现有配置优先级、热更新和诊断 API 语义一致。
- 提供迁移兼容层与文档，避免调用方一次性 break。
- 丰富文档：职责边界、目录约定、字段索引、迁移步骤、FAQ、示例导航。

**Non-Goals:**
- 不在本次引入远程配置中心或分布式控制平面。
- 不新增 CLI 诊断命令。
- 不改变业务执行语义（仅调整模块归属与接入方式）。

## Decisions

1. 新增全局 runtime 配置模块（建议：`runtime/config`）
- 决策：将 `Manager/Config/Snapshot` 等通用能力迁移到全局模块。
- 理由：职责语义准确，可供全局子系统复用。
- 备选：继续保留在 `mcp/runtime`。放弃原因：语义持续偏移，扩展成本上升。

2. 取消 `mcp/runtime` 并按功能拆分 MCP 子包
- 决策：将 MCP 语义拆分到 `mcp/profile`、`mcp/retry`、`mcp/diag`，不再保留 `mcp/runtime` 包。
- 理由：命名与职责一致，避免 “两个 runtime” 概念并存。

3. 新增全局 runtime diagnostics 模块
- 决策：将 `RecentCalls/RecentRuns/EffectiveConfigSanitized/RecentReloads` 等通用诊断 API 归属到全局 runtime 诊断层。
- 理由：避免 MCP 子域承载平台诊断入口，便于 runner/tool/skill/observability 复用。
- 备选：保持诊断 API 在 `mcp/runtime`。放弃原因：跨子系统调用语义不清晰。

4. 不保留旧路径兼容层
- 决策：直接移除旧包路径并在当前仓库内完成一次性迁移。
- 理由：避免概念混乱和后续长期兼容负担。
- 备选：保留转发层。放弃原因：用户明确不希望两个 runtime 并存。

5. 文档先行与一致性门禁
- 决策：在 README + docs 增加“模块边界与配置入口”规范，CI 增加文档一致性检查脚本（可先最小实现）。
- 理由：防止重构后文档漂移，降低团队沟通成本。

## Risks / Trade-offs

- [Risk] 迁移期双路径并存增加理解成本 -> Mitigation: 提供清晰 deprecation 注释与迁移映射表。
- [Risk] 包路径调整引发隐性编译断裂 -> Mitigation: 分批迁移 + 全量测试 + 示例更新。
- [Risk] 诊断 API 迁移导致字段或返回顺序漂移 -> Mitigation: 增加语义等价测试（字段集、排序、脱敏输出一致）。
- [Risk] 文档扩充带来维护负担 -> Mitigation: 加入文档一致性检查与 owner 责任归属。

## Migration Plan

1. 引入全局 runtime 配置/诊断包并复制现有实现。
2. 将 `mcp/http`、`mcp/stdio` 改为依赖新包接口（配置 + 诊断）。
3. 将 MCP 语义拆分到 `mcp/profile`、`mcp/retry`、`mcp/diag`。
4. 扩展 runner/tool/skill/observability 的配置与诊断读取接入点。
5. 更新 README 与 docs，并提供功能命名映射。
6. 验证：`go test ./...`、并发安全测试、文档一致性检查。

## Open Questions

- 新包最终路径是否使用 `runtime/config`（推荐）或 `internal/runtimecfg`（更强封装）需最终确认。
- 兼容层保留周期（例如 1~2 个 release）需在发布策略文档中明确。
