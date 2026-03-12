## 1. Security Config And Contracts

- [x] 1.1 在 `runtime/config` 增加 `security.scan.*` 与 `security.redaction.*` 配置字段与默认值
- [x] 1.2 增加安全配置校验（scan mode 枚举、关键词列表约束）并保持 `env > file > default`
- [x] 1.3 在 `core/types` 或等效契约层补充安全基线相关状态/错误分类（如需）

## 2. Unified Redaction Pipeline

- [x] 2.1 抽取统一 redaction 管线（关键词匹配基线 + 扩展口）
- [x] 2.2 将 `runtime/diagnostics` 的脱敏逻辑切换到统一管线并收敛存量差异
- [x] 2.3 将 `observability/event` 输出路径接入统一脱敏
- [x] 2.4 将 `context/assembler`（含 tail recap/stage payload）接入统一脱敏并修复存量泄漏路径

## 3. Quality Gate Security Integration

- [x] 3.1 在 Linux 质量门禁脚本接入 `govulncheck`（默认 strict）
- [x] 3.2 在 PowerShell 质量门禁脚本接入 `govulncheck`（默认 strict）
- [x] 3.3 更新 CI workflow，保证与本地脚本一致的 scan 语义与失败策略

## 4. Tests And Validation

- [x] 4.1 新增 redaction 单元测试（关键词命中、扩展关键词、非敏感字段不误伤）
- [x] 4.2 新增集成回归（diagnostics/event/assembler 路径敏感值不泄漏）
- [x] 4.3 执行并通过 `go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml` 与 `govulncheck`

## 5. Documentation Alignment

- [x] 5.1 更新 README 的安全基线章节与质量门禁命令
- [x] 5.2 更新 `docs/runtime-config-diagnostics.md` 的安全配置与脱敏行为说明
- [x] 5.3 更新 `docs/development-roadmap.md`（标注 Security S1 落地进展）并按需要补充 `docs/security-baseline.md`
