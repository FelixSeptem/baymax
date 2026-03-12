## Why

当前项目已具备多 provider、Context Assembler CA2 与运行时诊断能力，但安全基线尚未收敛为统一策略：依赖漏洞扫描未进入标准质量门禁，且脱敏逻辑分散在多个模块。现在落地 S1 安全基线，可以在不扩展 RBAC/审计平台范围的前提下，快速提升可上线性与合规可信度。

## What Changes

- 在质量门禁中引入 `govulncheck`，默认 strict（发现漏洞即失败），并覆盖 Linux、PowerShell 与 CI workflow。
- 新增统一安全配置（`security.scan.*`、`security.redaction.*`），支持扫描门禁策略与脱敏规则控制。
- 对 `runtime/diagnostics`、`observability/event`、`context/assembler` 的敏感信息处理进行一次性收敛，统一使用关键词脱敏管线。
- 对存量可见泄漏路径做集中修复，并为后续扩展保留规则扩展点。
- 同步更新 README 与 docs，确保文档和实现一致。

## Capabilities

### New Capabilities
- `security-baseline-s1`: 定义 S1 安全基线（扫描门禁 + 脱敏统一 + 配置可控）的行为契约。

### Modified Capabilities
- `runtime-config-and-diagnostics-api`: 扩展安全配置字段与诊断脱敏一致性要求。
- `go-quality-gate`: 扩展质量门禁要求，纳入 `govulncheck` strict 策略。

## Impact

- 受影响模块：`runtime/config`、`runtime/diagnostics`、`observability/event`、`context/assembler`、`scripts/*`、CI workflow。
- 质量门禁变化：在现有 `go test ./...`、`go test -race ./...`、`golangci-lint` 之外增加 `govulncheck`。
- 文档影响：`README.md`、`docs/runtime-config-diagnostics.md`、`docs/development-roadmap.md`（必要时新增 `docs/security-baseline.md`）。
