# 贡献与交付

## 分支生命周期

先同步最新 `master`，再创建语义化功能分支。proposal/design/spec/tasks、代码、测试和文档在同一分支完成；归档后合并、推送并删除已合并 worktree。

## 提交前检查

```powershell
openspec validate --all
pwsh -File scripts/check-openspec-example-impact-declaration.ps1
pwsh -File scripts/check-openspec-documentation-impact.ps1
pwsh -File scripts/check-docs-consistency.ps1
git diff --check
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml
```

质量门禁受环境影响时，必须在 PR 中记录命令、结果、风险、回滚点和未执行原因；不得把环境失败描述为通过。

## PR 必填

说明变更范围、验证命令、风险点、回滚点、文档影响、Example Impact Assessment 和未执行项。行为/配置/contract 变化必须包含测试与 canonical 文档路径。
