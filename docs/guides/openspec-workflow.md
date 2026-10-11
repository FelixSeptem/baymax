# OpenSpec 工作流

1. `openspec list --json` 查看 active changes。
2. 从最新 `master` 创建语义化分支。
3. 创建 `proposal.md`、`design.md`、`tasks.md`，行为/配置/contract 变化再维护 `specs/*/spec.md`。
4. 在三份提案文档中添加 [Documentation Impact Assessment](../documentation-style-guide.md)，逐项填写 architecture、components、configuration、contract/API、diagnostics、examples、CLI/integration、best practices、roadmap。
5. 三份工件都填写 [Public Semantic Documentation Commitment](../documentation-style-guide.md)：对外语义变化填写 `是` 并在同一 change 更新维护文档、owner 与验证证据；填 `否` 时给出具体理由，三份声明须一致。
6. 同时填写 Example Impact Assessment（`新增示例`、`修改示例` 或带理由的 `无需示例变更（附理由）`）。
7. 按 tasks 实施，任务必须同时有代码/测试/文档/门禁证据；每完成一项立即勾选。
8. 运行 `openspec validate --all`、文档影响、Example Impact、roadmap 和质量门禁。
9. 使用 `scripts/openspec-archive-seq.ps1` 归档，禁止手工重命名 archive；随后合并最新基准并推送。

## Documentation Impact Assessment 规则

Outcome 只能是 `新增文档`、`修改文档`、`无需文档变更（附理由）`。前两者必须写 affected paths、owner、verification；无影响必须说明为什么变更面不触及该主题。行为、配置、contract、诊断或示例路径发生变化却声明无影响时，漂移门禁阻断。
