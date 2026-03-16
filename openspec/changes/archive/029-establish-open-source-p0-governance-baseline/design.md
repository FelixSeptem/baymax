## Context

当前仓库已具备功能与质量基线（test/race/lint/govulncheck），但开源协作面仍缺少“对外可预期”的治理资产：
- 版本与兼容承诺未形成稳定文档入口；
- 安全漏洞报告路径未标准化（需 GitHub Security Advisory）；
- 社区贡献入口与评审清单不完整；
- CI workflow 在开源可维护性上仍有优化空间（工具版本漂移、步骤重复、权限未显式最小化）。

该提案聚焦文档与流程治理，不触及 runtime 代码语义。

## Goals / Non-Goals

**Goals:**
- 建立开源 P0 最小治理闭环：发布兼容承诺 + 安全响应入口 + 贡献评审流程。
- 将治理规则沉淀为仓库内可执行资产（文档、模板、CI 规则）。
- 保持 README/docs/CI 单一口径一致，降低外部接入与维护成本。

**Non-Goals:**
- 不新增运行时功能，不修改 core/runner/mcp/model/context 等行为。
- 不引入 SBOM/license scan 自动化（保留后续迭代）。
- 不做跨仓库发布自动化与 tag 策略变更。

## Decisions

### Decision 1: P0 范围一次性收敛
- 方案：单提案同时交付 P0-1/P0-2/P0-3。
- 原因：三者耦合度高，拆分会造成文档与流程短期不一致。
- 备选：拆成 3 个小提案；否决原因是维护成本更高、对外窗口更长。

### Decision 2: 安全入口采用 GitHub Security Advisory
- 方案：`SECURITY.md` 明确使用 GitHub 私密披露流程。
- 原因：与开源协作生态对齐，避免公开 issue 泄露漏洞细节。
- 备选：邮箱流程；否决原因是审计追踪与协作透明度较弱。

### Decision 3: CI 工作流仅做治理增强，不改质量门禁语义
- 方案：固定 linter 版本、去除重复 hygiene step、增加 `permissions` 与 `timeout-minutes`。
- 原因：提升可复现性与安全性，同时避免改变现有 test/race/lint/govulncheck 判定口径。
- 备选：重构为多 job matrix；本期否决，超出 P0 范围。

### Decision 4: CODE_OF_CONDUCT 纳入本期
- 方案：新增 `CODE_OF_CONDUCT.md` 并在贡献文档中引用。
- 原因：为社区协作建立行为边界，减少治理真空。
- 约束：后续治理调整需同步更新该文档。

## Risks / Trade-offs

- [Risk] 文档新增较多，可能出现引用漂移或重复表述
  -> Mitigation: 在 README 与 roadmap 保留单一入口并通过 docs consistency 脚本校验关键引用。

- [Risk] 固定 linter 版本后可能滞后于上游修复
  -> Mitigation: 在 roadmap 中保留周期性升级任务，按月或按 release 批量更新。

- [Risk] 模板规则过严增加外部贡献摩擦
  -> Mitigation: 模板保持最小必填字段，只强制质量与兼容性相关检查项。

## Migration Plan

1. 新增治理文档与模板文件（不影响代码编译与测试）。
2. 更新 CI workflow 的治理项（版本固定、权限、超时、去重）。
3. 同步 README 与 roadmap 引用，确保文档入口一致。
4. 执行质量门禁（`go test ./...`、`go test -race ./...`、`golangci-lint`）验证无行为回归。
5. 在提案验收后归档并更新 archive 索引。

回滚策略：
- 若治理文档或模板引发协作阻塞，可独立回滚对应文件，不影响 runtime 功能路径。
- 若 CI 调整导致误拦截，可先回退 workflow 到前一版本并保留文档资产。

## Open Questions

- `docs/versioning-and-compatibility.md` 是否在后续提案中扩展为“跨 provider 兼容矩阵自动生成”？（本期仅手工文档化）
- 是否在下个阶段纳入 `SECURITY.md` 的 SLA 指标可视化与 issue label 自动化？
