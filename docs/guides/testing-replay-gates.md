# 测试、Replay 与 Gate

## 测试层次

1. package unit：覆盖正向、负向和边界。
2. integration：覆盖 adapter、Run/Stream parity 和生命周期。
3. contract：验证 schema、错误分类、配置优先级和边界。
4. replay：使用固定 fixture 验证诊断和终态确定性。
5. gate：离线检查文档、roadmap、OpenSpec、示例和仓库卫生。

## 常用命令

```powershell
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml
pwsh -File scripts/check-docs-consistency.ps1
pwsh -File scripts/check-openspec-documentation-impact.ps1
pwsh -File scripts/check-quality-gate.ps1
```

完整 contract 索引见 [mainline-contract-test-index.md](../mainline-contract-test-index.md)。Windows 文件占用时，按贡献指南使用隔离 `GOCACHE`/`GOTMPDIR` 的逐包验证，并记录未执行项。

## 回滚

fixture 或 gate 失败时先回滚变更分支上的文档/脚本，不修改生产配置；确认事实源后再更新断言。
