# 贡献指南（Contributing）

## 开发准备

前置条件：

- Go 1.26+
- golangci-lint
- govulncheck（建议安装；质量门禁默认使用）

安装依赖：

```bash
go mod tidy
```

## 本地质量检查

Linux/macOS：

```bash
bash scripts/check-quality-gate.sh
```

Windows PowerShell：

```powershell
pwsh -File scripts/check-quality-gate.ps1
```

提交 PR 前至少执行：

```bash
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml
```

## Pull Request 流程（必填）

1. 保持变更聚焦，避免无关改动混入。
2. 行为/配置/契约变化必须同步更新文档。
3. 行为变化必须补充或更新测试。
4. 在 PR 模板中完整填写必填段落与检查项。
5. CI 通过后再请求评审。

PR 模板为中文优先，接受英文内容；但结构与必填检查项必须完整。

默认 CI 在 pull request 事件执行 `contribution-template-gate`。建议在分支保护中将其设为 required status check。

## 评审检查清单（Required）

- 测试已更新，或给出不更新的理由。
- 用户可见变化已同步文档。
- 变更影响（含 breaking 风险）已明确说明。
- 如有迁移影响，已在 `CHANGELOG.md` 或 PR 说明中标注。

## Community Conduct

本项目遵循 `CODE_OF_CONDUCT.md`。
参与贡献即表示同意遵守该规范。

## 安全报告

漏洞报告请使用 `SECURITY.md` 中的邮箱私报流程（非公开 issue）。

## 治理口径说明

- 当前处于 pre-1.x 阶段，不提供兼容性承诺。
- 维护范围为最新 minor 线，旧 minor 回溯修复为 best-effort。
