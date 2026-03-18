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
