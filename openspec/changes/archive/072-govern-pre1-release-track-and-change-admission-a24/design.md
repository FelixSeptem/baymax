## Context

当前仓库在能力层面已覆盖多代理主链路与外部 adapter conformance，但版本治理仍处于 `0.x`。近期 roadmap 口径调整表明：如果缺少可执行约束，文档很容易在“继续 pre-1 迭代”与“提前承诺 1.0 稳定”之间反复漂移，影响提案优先级与外部预期管理。

A24 不引入 runtime 新能力，而是把 `0.x` 阶段治理收敛为可验证规则，避免 roadmap 与版本策略分离演进。

## Goals / Non-Goals

**Goals:**
- 固化 `0.x` 阶段发布口径：显式声明不做 `1.0/prod-ready` 承诺。
- 定义可执行的提案准入标准，限制“无边界新增提案”。
- 把 pre-1 口径一致性纳入 docs consistency 与质量门禁阻断路径。
- 保持 README、roadmap、versioning 三处口径同步可追溯。

**Non-Goals:**
- 不修改 runtime 行为、API、配置语义。
- 不新增平台化控制面能力。
- 不在 A24 内引入发布自动化系统或发行流水线改造。

## Decisions

### 1) 使用“文档契约 + 门禁脚本 + contributioncheck”三层治理
- 方案：在 OpenSpec 定义治理契约；通过 `check-docs-consistency.*` 与 `tool/contributioncheck` 执行阻断。
- 原因：与现有工程习惯一致，改动最小且可回归。
- 备选：仅文档约定。拒绝原因：不可执行，易漂移。

### 2) 准入规则采用“类别约束 + 必填信息”双门槛
- 方案：新增提案需满足目标类别（契约/可靠性/门禁/DX）并填写 `Why now`、风险、回滚、文档影响、验证命令。
- 原因：既控制范围，又保证评审可操作性。
- 备选：只做类别约束。拒绝原因：评审信息常不完整，后续追踪成本高。

### 3) 以“pre-1 口径一致性”作为 docs gate 阻断项
- 方案：在 docs consistency 检查中加入版本阶段断言，要求 roadmap/versioning/README 不出现冲突表述。
- 原因：版本阶段属于对外核心信号，应 fail-fast。
- 备选：release 前人工检查。拒绝原因：容易漏检且不可复现。

## Risks / Trade-offs

- [Risk] 文档约束过严导致正常文案调整频繁触发失败  
  → Mitigation: 采用关键语义断言而非逐字匹配，减少非语义噪声。

- [Risk] 准入规则增加提案编写负担  
  → Mitigation: 提供最小模板字段，避免冗长流程化文档。

- [Risk] README 与 roadmap 的状态更新节奏不一致  
  → Mitigation: 在同一 PR 要求同步更新，并由 docs consistency gate 阻断。

## Migration Plan

1. 在 roadmap 中固化 `0.x` 阶段定位与准入规则。
2. 更新 versioning 文档，使其与 roadmap 口径一致。
3. 扩展 docs consistency 与 contributioncheck，新增 pre-1 口径一致性断言。
4. 更新主 README 进度快照与文档入口说明（如需要）。
5. 将检查接入质量门禁标准路径并阻断失败。

回滚策略：
- 若 gate 初期噪声过大，可先回滚新增断言并保留文档口径；
- 不影响 runtime 主链路功能与行为。

## Open Questions

- 当前无阻塞问题，按推荐值执行：维持 `0.x`、不做 `1.0/prod-ready` 承诺、启用 docs/gate 阻断一致性检查。
