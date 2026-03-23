## Context

当前主线进入 A36 实施阶段后，仓库已从“能力扩展优先”转入“质量收敛优先”。  
现状暴露的关键问题不是功能缺失，而是门禁可信度不一致：

- PowerShell 门禁脚本对 native command 非零退出传播不统一，存在“测试失败但脚本仍继续并打印通过”的风险。
- 文档状态口径（README/roadmap）与 OpenSpec authority（`openspec list --json` + `archive/INDEX.md`）已出现实证漂移。
- Linux shell 门禁与 PowerShell 门禁在失败传播语义上不完全等价，削弱 cross-platform gate parity。

该提案目标是将门禁行为收敛为 deterministic fail-fast，并把状态口径漂移治理纳入同一阻断路径。

## Goals / Non-Goals

**Goals:**
- 统一关键 PowerShell 门禁脚本的 native command 失败传播语义为 fail-fast（非零即阻断）。
- 保证 shell 与 PowerShell 在质量门禁上的 pass/fail 语义等价。
- 修复 docs consistency 路径的“假阳性”风险，并强制状态口径漂移阻断。
- 为失败传播语义增加可回归的契约测试，防止后续脚本回退到宽松模式。

**Non-Goals:**
- 不新增运行时业务能力，不修改 Runner/Composer/Scheduler 业务语义。
- 不调整现有质量门禁内容清单（测试项本身不扩容到平台化范围）。
- 不引入平台化控制面或外部依赖编排系统。

## Decisions

### 1) 引入统一 PowerShell native 执行包装并强制关键脚本使用
- 决策：新增统一的 strict-native 执行包装（例如 `Invoke-NativeStrict`），封装命令执行、`$LASTEXITCODE` 判定与错误信息格式化。
- 原因：PowerShell 的 native command 非零退出不会自动触发 `catch`，仅依赖 `$ErrorActionPreference='Stop'` 不足以保证阻断。
- 备选：在每个命令后手写 `$LASTEXITCODE` 校验。拒绝原因：重复且易漏，后续维护成本高。

### 2) 收口关键 gate 脚本到一致失败传播语义
- 决策：优先覆盖 `check-quality-gate.ps1`、`check-docs-consistency.ps1`、`check-multi-agent-shared-contract.ps1` 与 adapter/security 子 gate 脚本。
- 原因：这些脚本位于主阻断路径，任何失败传播漏洞都会放大为仓库级假阳性。
- 备选：只修一个脚本。拒绝原因：无法系统消除语义漂移，仍会在其他脚本复发。

### 3) 保留唯一“可告警不阻断”例外为 `govulncheck warn`
- 决策：仅在已有治理策略下保留 `BAYMAX_SECURITY_SCAN_MODE=warn` 的告警放行；其余 native command 一律 fail-fast。
- 原因：与现有质量策略一致，避免引入新的隐式豁免。
- 备选：扩展更多 warn 白名单。拒绝原因：会稀释门禁可信度与语义清晰度。

### 4) 将状态口径漂移与脚本失败传播纳入同一契约面
- 决策：在 `release-status-parity-governance` 与 `go-quality-gate` 同时补充要求，确保“口径冲突”与“脚本未阻断”都可被契约测试覆盖。
- 原因：只改代码不改契约会导致后续反复回归；只改契约不改脚本无法止血。
- 备选：仅更新 README/roadmap。拒绝原因：不能防止同类问题再次出现。

## Risks / Trade-offs

- [Risk] 严格失败传播会让历史上“可忽略失败”立刻暴露，短期内可能增加 CI 失败率  
  -> Mitigation: 明确唯一例外策略（govulncheck warn）并在 proposal/tasks 中固定范围。

- [Risk] 包装函数引入后，脚本若绕过包装仍可能复发  
  -> Mitigation: 增加 contributioncheck 静态契约测试，检测关键脚本是否存在未受控 native 调用。

- [Risk] 文档状态在并行 session 下仍可能短时漂移  
  -> Mitigation: 将 status parity 校验纳入 docs consistency 阻断，并在 roadmap 维护流程中明确 authority 源。

## Migration Plan

1. 新增 PowerShell strict-native 执行包装与统一错误输出约定。
2. 批量替换关键 `check-*.ps1` 脚本中的 native command 调用路径，确保非零立即阻断。
3. 修复 `check-docs-consistency.ps1` 的失败传播路径，确保 status parity 失败时脚本返回非零。
4. 更新 README/roadmap 的 active/archived 口径到 authority 当前状态。
5. 增加/更新 gate 契约测试与索引文档，覆盖失败传播与状态口径收敛语义。
6. 执行质量门禁与 OpenSpec 严格校验，确认变更可实施。

回滚策略：
- 可回滚到旧脚本实现（不涉及运行时数据迁移）；
- 但回滚前需确认不会重新引入“假阳性通过”风险。

## Open Questions

无阻塞待确认项，按推荐值执行：
- 默认严格失败传播：开启
- 例外策略：仅 `govulncheck warn`
- 覆盖范围：关键 gate 脚本全覆盖
