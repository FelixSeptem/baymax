# 文档写作与页面模板

Baymax 文档中文为主，代码标识、API、协议名、配置键和错误码保留英文。页面应面向读者任务组织，而不是按提交历史组织。

## 术语约定

| 中文 | 英文保留形式 | 说明 |
| --- | --- | --- |
| 运行时 | runtime | `runtime/*` 的执行与状态边界 |
| 提供方 | Provider | 模型协议适配层 |
| 工具 | tool | 本地工具或 MCP 工具 |
| 诊断 | diagnostics | 可查询、可回放的运行事实 |
| 回放 | replay | 使用 fixture 重建可验证事实 |
| 门禁 | gate | 离线、确定性的检查入口 |
| 文档影响评估 | Documentation Impact Assessment | OpenSpec 必填治理段落 |

禁止用“引擎”“万能代理”等含义不明确的营销词代替模块名称；Run 和 Stream 在语义等价处使用同一术语。

## 组件页面模板

每个 `docs/components/*.md` 页面至少包含：

1. Purpose：组件解决的问题和明确非目标。
2. Owned facts/state：组件拥有或派生的事实、状态和生命周期。
3. Inputs/outputs：输入、输出、错误和边界条件。
4. Dependencies and forbidden dependencies：允许与禁止的依赖方向。
5. Lifecycle/data flow：关键路径和 Run/Stream 对等性。
6. Failure and rollback：失败分类、原子回滚和停用方式。
7. Observability：诊断字段和 `RuntimeRecorder` 写入责任。
8. Verification：可执行命令或明确的 bounded 验证范围。
9. Source/spec/test/example links：事实源、测试和示例链接。

## 链接、弃用与审查

- 仓库内链接使用相对路径；链接目标必须在同一提交中存在。
- 旧页面使用“已迁移”短 stub，不复制完整内容。
- 每个页面注明维护 owner；涉及行为变化时注明 rollback。
- 文档变更必须运行 `scripts/check-docs-consistency.*` 和 `scripts/check-openspec-documentation-impact.*`。
- 页面不能用提案编号命名；OpenSpec 目录是唯一允许保留提案标号的例外。

## 文档影响评估模板

新提案的 `proposal.md`、`design.md`、`tasks.md` 都必须包含以下公开语义文档同步承诺。它不是对文档影响表的替代，而是每个 OpenSpec 对“对外能力变化必须同步维护文档”的明确接受：

```markdown
## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`docs/guides/integration.md`
- 维护 owner：maintainer
- 验证证据：`pwsh -File scripts/check-docs-consistency.ps1`
```

没有对外语义变化时，必须写明理由：

```markdown
## Public Semantic Documentation Commitment

- 对外语义变化：否
- 文档同步状态：无需同步（附理由）：本 change 只修改内部测试工具，不改变公开 API、运行时行为、配置、诊断、CLI/integration 或示例语义。
- 维护文档：—
- 维护 owner：maintainer
- 验证证据：`pwsh -File scripts/check-openspec-documentation-impact.ps1`
```

“是”必须在同一 change 中提交维护文档，列出的每个路径必须存在并出现在功能分支相对 `master` 的 Git 差异中（包括未提交、未跟踪文件）；仅引用既有文档不算同步。“否”必须给出具体理由。三份工件的承诺不能互相矛盾，门禁会检查缺失、非法值和文档同步漂移。

新提案的 `proposal.md`、`design.md`、`tasks.md` 还必须包含以下文档影响表（可复制后填写）：

```markdown
## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | maintainer | reason: ... |
| components | 无需文档变更（附理由） | — | maintainer | reason: ... |
| configuration | 无需文档变更（附理由） | — | maintainer | reason: ... |
| contract/API | 无需文档变更（附理由） | — | maintainer | reason: ... |
| diagnostics | 无需文档变更（附理由） | — | maintainer | reason: ... |
| examples | 无需文档变更（附理由） | — | example owner | reason: ... |
| CLI/integration | 无需文档变更（附理由） | — | integration owner | reason: ... |
| best practices | 无需文档变更（附理由） | — | maintainer | reason: ... |
| roadmap | 修改文档 | docs/development-roadmap.md | release owner | check-openspec-roadmap-status-consistency |
```

Outcome 只能是 `新增文档`、`修改文档` 或 `无需文档变更（附理由）`。前两者必须填写路径、owner 和验证命令；无影响必须写出理由，不能只写 `—`。
