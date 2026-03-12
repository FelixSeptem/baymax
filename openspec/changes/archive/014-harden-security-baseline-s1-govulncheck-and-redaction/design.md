## Context

仓库当前质量门禁由 `go test`、`go test -race`、`golangci-lint` 组成，尚未把依赖漏洞扫描作为强制项；同时脱敏策略已存在但散落在 diagnostics、event 记录、context assembler 等路径，规则不完全统一。用户确认本提案范围为 S1 基线：默认 strict 扫描、一次性收敛存量泄漏路径、关键词匹配策略先行并预留扩展，不包含 RBAC 或外部告警平台接入。

## Goals / Non-Goals

**Goals:**
- 将 `govulncheck` 接入标准质量门禁，默认 strict。
- 新增可配置安全策略（扫描与脱敏），保证 `env > file > default`。
- 统一三个关键路径的脱敏行为：`runtime/diagnostics`、`observability/event`、`context/assembler`。
- 通过单元/集成测试锁定“敏感键不出现在日志/诊断/recap 输出”基线。
- 同步 README/docs，使行为与文档一致。

**Non-Goals:**
- 不实现 RBAC、租户级权限系统。
- 不实现完整内容审核平台或外部 SIEM/告警平台集成。
- 不引入复杂 DLP 引擎（本期仅关键词规则 + 扩展口）。

## Decisions

### Decision 1: 安全扫描优先采用 govulncheck 并默认 strict
- Choice: 在 Linux 脚本、PowerShell 脚本、CI workflow 中统一执行 `govulncheck`，默认失败即阻断。
- Rationale: 与 Go 生态原生工具链一致，落地成本最低，结果可复现。
- Alternative: 先接入 `gosec` 作为唯一门禁。Rejected：规则噪音与误报处理成本较高，留作后续增强。

### Decision 2: 安全配置并入 runtime 配置体系
- Choice: 增加 `security.scan.*`、`security.redaction.*` 配置并复用现有 config manager 的优先级与热更新机制。
- Rationale: 避免新增并行配置通道，保证运行时行为可审计。
- Alternative: 使用独立安全配置文件。Rejected：会增加配置漂移风险。

### Decision 3: 脱敏管线集中化
- Choice: 抽取统一 redaction 规则入口，由 diagnostics/event/assembler 复用，关键词匹配为默认策略。
- Rationale: 一次性收敛存量差异，降低后续维护成本。
- Alternative: 各模块保留本地规则。Rejected：不可持续，易出现遗漏。

### Decision 4: 关键词策略先行并预留扩展口
- Choice: 基线关键词包含 `token/password/secret/api_key/apikey`，接口允许后续扩展自定义词典或策略插件。
- Rationale: 与当前项目规模和交付节奏匹配。
- Alternative: 直接接入模型/规则混合检测。Rejected：超出本提案范围。

## Risks / Trade-offs

- [Risk] strict 模式可能因上游漏洞波动导致 CI 不稳定。 -> Mitigation: 输出清晰失败信息与最小可操作修复建议，保留受控白名单配置位（默认空）。
- [Risk] 关键词匹配可能存在漏检。 -> Mitigation: 在统一入口预留扩展接口与回归测试样例池。
- [Risk] 一次性收敛存量路径可能引入行为变化。 -> Mitigation: 增加兼容测试，确保仅敏感值被掩码，不改变非敏感字段语义。

## Migration Plan

1. 扩展 runtime 安全配置模型与默认值。
2. 抽取统一 redaction pipeline 并替换 diagnostics/event/assembler 原有分散逻辑。
3. 接入 `govulncheck` 到本地脚本与 CI workflow（strict）。
4. 补齐安全基线测试与文档。

Rollback strategy:
- 允许通过配置将扫描策略降级为 warn（仅应急）；默认仍为 strict。
- 保持旧字段兼容，但统一由新 pipeline 执行脱敏。

## Open Questions

- 是否需要在 S2 引入 `gosec` 规则集并按风险等级分层门禁。
- 是否需要在后续阶段引入可配置白名单（CVE allowlist）治理流程。
