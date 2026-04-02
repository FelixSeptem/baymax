# skill/loader 组件说明

## 功能域

`skill/loader` 负责技能发现、触发评分与 bundle 组装：

- 发现：扫描仓库 `AGENTS.md` 中技能声明
- 评分：显式触发 + 语义触发（词法/embedding）
- 组装：生成 system prompt 片段、启用工具集合与 workflow hints

## 架构设计

`Loader` 通过两个阶段工作：

- `Discover`：解析技能路径、元数据、触发词与优先级
- `Compile`：基于输入选择技能，读取技能文件并编译为 `types.SkillBundle`

评分策略来自 `runtime/config`：

- `lexical_weighted_keywords`
- `lexical_plus_embedding`（需注入 embedding scorer）
- 固定/自适应预算（`fixed|adaptive`）

该模块会发射 `skill.discovered` / `skill.loaded` / `skill.warning` 事件供 RuntimeRecorder 聚合。

## 关键入口

- `loader.go`

## 边界与依赖

- 仅处理技能元数据与编译，不直接执行工具调用。
- 通过 `types.SkillLoader` 契约与上层解耦，避免耦合具体编排器。
- 评分策略变更需同步配置校验、文档与契约测试。

## 配置与默认值

- 默认评分策略与预算由 `runtime/config` 的 skill trigger 子域提供。
- 未注入 embedding scorer 时会降级到词法评分路径并记录原因码。
- 默认预算策略支持 `fixed` 与 `adaptive`，按配置生效。

A65 新增与 loader 协作的 runtime skill 子域（由 runner 在 Run/Stream 前统一接线）：

- discovery：
  - `runtime.skill.discovery.mode=agents_md|folder|hybrid`
  - `runtime.skill.discovery.roots`（`folder|hybrid` 模式必填）
- preprocess：
  - `runtime.skill.preprocess.enabled`
  - `runtime.skill.preprocess.phase=before_run_stream`
  - `runtime.skill.preprocess.fail_mode=fail_fast|degrade`
- bundle mapping：
  - `runtime.skill.bundle_mapping.prompt_mode=disabled|append`
  - `runtime.skill.bundle_mapping.whitelist_mode=disabled|merge`
  - `runtime.skill.bundle_mapping.conflict_policy=fail_fast|first_win`

约束：

- preprocess 失败在 `fail_fast` 下直接终止，在 `degrade` 下继续执行并写入 warning + reason。
- `SkillBundle -> tool whitelist` 映射受 sandbox/allowlist 上界约束，超界项不会生效。
- Run/Stream 共享同一 preprocess 与 mapping 路径，禁止单入口语义分叉。

## 可观测性与验证

- 关键验证：`go test ./skill/loader -count=1`。
- 关键观测事件：`skill.discovered`、`skill.loaded`、`skill.warning`。
- 与 runtime config 联动验证需覆盖评分权重、预算阈值和回滚语义。

## 扩展点与常见误用

- 扩展点：新增 scorer、扩展 tokenization 策略、增加 skill metadata 字段。
- 常见误用：在 loader 层执行工具调用，破坏“发现/编译”职责边界。
- 常见误用：embedding 故障后直接失败而不走降级策略，导致可用性下降。
