## 1. P0-1 发布与兼容承诺

- [x] 1.1 新增 `docs/versioning-and-compatibility.md`，定义 SemVer、breaking change 规则、Go 支持窗口与 provider 支持级别
- [x] 1.2 新增 `CHANGELOG.md` 模板并在 README 增加版本/兼容入口链接
- [x] 1.3 对齐 `README.md` 与 `docs/development-roadmap.md` 的开源 P0 口径，确保单一事实源

## 2. P0-2 安全响应入口

- [x] 2.1 新增 `SECURITY.md`，明确 GitHub Security Advisory 为私密披露渠道
- [x] 2.2 在 `SECURITY.md` 补齐响应时限、分级处理与披露流程
- [x] 2.3 在 README 中增加安全报告入口并避免与现有流程冲突

## 3. P0-3 贡献与评审最小闭环

- [x] 3.1 新增 `CONTRIBUTING.md`（开发准备、测试命令、提交流程、文档同步要求）
- [x] 3.2 新增 `CODE_OF_CONDUCT.md`，并在贡献文档中引用
- [x] 3.3 新增 GitHub 模板：`bug_report`、`feature_request`、`pull_request_template`，包含最小评审清单

## 4. CI 治理增强（不改门禁语义）

- [x] 4.1 更新 `.github/workflows/ci.yml`：固定 `golangci-lint` 版本（替代 `latest`）
- [x] 4.2 去除与 `scripts/check-quality-gate.sh` 重复的 repository hygiene 步骤
- [x] 4.3 增加 workflow 最小 `permissions` 与 `timeout-minutes`

## 5. 验证与文档一致性

- [x] 5.1 执行 `go test ./...` 与 `go test -race ./...`，确认功能行为无回归
- [x] 5.2 执行 `golangci-lint run --config .golangci.yml`
- [x] 5.3 执行 `pwsh -File scripts/check-docs-consistency.ps1`，确保 README/docs 引用一致
