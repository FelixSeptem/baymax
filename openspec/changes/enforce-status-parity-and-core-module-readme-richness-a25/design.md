## Context

仓库当前以 OpenSpec 作为进度权威来源，但 roadmap/README 仍由人工摘要维护，存在状态滞后风险。与此同时，核心模块 README 已建立统一骨架（功能域/架构设计/关键入口/边界与依赖），但在配置、诊断、扩展点和验证入口上信息深度不足，外部贡献者仍需要回溯源码或分散文档。

A25 在 `0.x` 阶段聚焦文档治理，不变更 runtime 行为。目标是把状态对齐与模块文档深度都变成可执行 gate。

## Goals / Non-Goals

**Goals:**
- 建立 `openspec -> roadmap/README` 状态一致性检查，减少人工快照漂移。
- 定义核心模块 README 丰富化基线（必填段落 + 最小可执行信息）。
- 将上述两类检查接入 docs consistency 与 quality gate 阻断路径。
- 维持跨平台一致性（shell/PowerShell 同语义）。

**Non-Goals:**
- 不新增 runtime API 或配置字段。
- 不引入平台化控制面能力。
- 不替代详细 API 文档；A25 只定义模块 README 的最小深度基线。

## Decisions

### 1) 状态一致性采用“权威源比对”而非硬编码版本号
- 方案：以 `openspec list --json` 和 `openspec/changes/archive/INDEX.md` 为权威源，比对 roadmap/README 的状态叙述。
- 原因：避免每次提案手工维护静态常量导致脚本过快失效。
- 备选：固定检查最新 A 编号。拒绝原因：脆弱且维护成本高。

### 2) README 丰富化采用“必填章节 + 允许 N/A”策略
- 方案：核心模块 README 必须包含统一章节集合，章节不适用时需显式写明 `N/A` 或“当前不适用”。
- 原因：保证信息结构可预测，同时避免对不同模块强行同质化细节。
- 备选：只检查文件存在。拒绝原因：无法保证可用信息密度。

### 3) 先接入 docs consistency，再由 quality gate 统一阻断
- 方案：新增断言放在 `check-docs-consistency.*` 与 `tool/contributioncheck`，由 `check-quality-gate.*` 调用阻断。
- 原因：复用既有治理路径，降低新门禁接入风险。
- 备选：新增独立 gate 脚本。拒绝原因：门禁分散，维护负担上升。

### 4) 核心模块范围使用显式清单
- 方案：A25 首期覆盖以下 README：
  - `a2a/README.md`
  - `core/runner/README.md`
  - `core/types/README.md`
  - `tool/local/README.md`
  - `mcp/README.md`
  - `model/README.md`
  - `context/README.md`
  - `orchestration/README.md`
  - `runtime/config/README.md`
  - `runtime/diagnostics/README.md`
  - `runtime/security/README.md`
  - `observability/README.md`
  - `skill/loader/README.md`
- 原因：范围清晰、可验证、可评审。
- 备选：扫描所有目录自动发现。拒绝原因：容易引入噪声和误报。

## Risks / Trade-offs

- [Risk] 状态对齐规则过严，正常文案改写触发误报  
  → Mitigation: 采用语义标记检查（状态/变更名/阶段），避免逐字匹配。

- [Risk] README 统一模板导致表达僵化  
  → Mitigation: 仅约束章节存在与最低信息项，不约束段落长度和叙事风格。

- [Risk] 维护者更新成本增加  
  → Mitigation: 提供可复制的模块 README 样板段落与 check failure 指引。

## Migration Plan

1. 在 contributioncheck 中新增 status parity 与 module README richness 校验器。
2. 扩展 `check-docs-consistency.sh/.ps1` 接入上述校验。
3. 确保 `check-quality-gate.sh/.ps1` 继续通过 docs consistency 阻断。
4. 批量补齐核心模块 README 必填段落。
5. 更新 roadmap/README 状态快照与 mainline contract index 映射。

回滚策略：
- 若误报过高，可临时回滚新增断言并保留 README 内容增量；
- 不影响 runtime 执行路径。

## Open Questions

- 当前无阻塞问题，按推荐值执行：显式核心模块清单、必填章节 + N/A、status parity fail-fast 阻断。
