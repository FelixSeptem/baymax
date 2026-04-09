## Context

`examples/agent-modes` 在 a62 中完成了统一目录、矩阵与门禁接线，当前已具备可运行入口。但代码层仍存在“同构模板实现”问题：多数模式通过统一的本地模型/工具骨架输出结构化结果，未充分体现模式语义差异与真实主干链路组合。该状态可用于基础冒烟，但不足以作为生产迁移参考。

本提案（a71）独立于 a62，目标是把示例从“可运行”提升到“可迁移的真实语义样例”，并通过新门禁阻断回退到模板化占位实现。

约束：
- 保持 `library-first`，不引入平台化控制面。
- 不重定义既有 contract 语义；示例只复用既有契约口径。
- shell/PowerShell 门禁保持 parity。

## Goals / Non-Goals

**Goals:**
- 对 `examples/agent-modes` 的 28 个模式完成真实逻辑替换，覆盖 `minimal + production-ish` 双变体。
- 为每个模式建立真实语义锚点（模式专属 runtime path + 可验证输出证据）。
- 新增“真实语义门禁 + README 同步门禁”，阻断模板回流与文档漂移。
- 将模式级能力映射到 `contract/gate/replay`，形成可审计闭环。

**Non-Goals:**
- 不引入新平台组件（控制面、托管执行面、多租户运维面板）。
- 不在 a71 中扩展新的业务 contract 范畴（仅复用现有 contract）。
- 不要求每个模式都触达全部域（mcp/model/context/memory 等按模式职责覆盖）。

## Decisions

### Decision 1: 以“模式语义锚点”替代“统一模板骨架”
- 方案：每个模式必须声明并实现至少一个模式专属语义锚点，例如：
  - `rag-hybrid-retrieval`：检索候选构建 + 重排 + fallback 证据；
  - `mcp-governed-stdio-http`：传输策略选择 + failover 决策证据；
  - `state-session-snapshot-recovery`：快照导出/恢复一致性证据。
- 原因：统一模板难以证明模式差异与迁移价值。
- 备选：继续用统一模板，仅增强输出字段。
- 取舍：实现成本更高，但示例可用性和审计价值显著提升。

### Decision 2: 真实示例判定采用“双条件”
- 方案：示例通过必须同时满足：
  1) 命中主干真实运行时路径（按模式至少一个域）；
  2) 输出模式语义验证证据（非固定占位字段）。
- 原因：仅看 import 无法证明行为真实，仅读输出也可能伪造。
- 备选：只做静态代码扫描。
- 取舍：加入运行证据校验，准确性更高但门禁实现更复杂。

### Decision 3: README 同步由门禁强制而非人工约定
- 方案：当 `main.go` 行为变更时，必须同步更新同目录 README；README 必须包含 `Run`、`Prerequisites`、`Real Runtime Path`、`Expected Output/Verification`、`Failure/Rollback Notes`。
- 原因：示例交付入口以 README 为准，人工约束易漂移。
- 备选：仅在 PR 模板提示。
- 取舍：文档维护成本提升，但交付一致性更可靠。

### Decision 4: 分批替换策略采用“先高价值模式，再全量覆盖”
- 方案：按批次推进：
  1) P0 模式（rag/structured-output/mcp/hitl/context/sandbox/realtime/multi-agents）；
  2) P1 编排与治理模式；
  3) P2 主干流程与 adapter 模式。
- 原因：先落地高价值路径，尽早形成验证模型与门禁基线。
- 备选：28 模式一次性替换。
- 取舍：分批更稳健，但总周期略长。

### Decision 5: 将 a71 门禁接入 go-quality-gate 且默认阻断
- 方案：新增：
  - `check-agent-mode-real-runtime-semantic-contract.sh/.ps1`
  - `check-agent-mode-readme-runtime-sync-contract.sh/.ps1`
  并接入 `check-quality-gate.*`。
- 原因：防止“提案写了，后续回流模板实现”。
- 备选：仅在 a71 阶段临时执行。
- 取舍：长期阻断提升稳定性，但 CI 耗时增加。

## Risks / Trade-offs

- [Risk] 示例改造范围大，短期改动量高。  
  -> Mitigation: 按 P0/P1/P2 分批推进，批次内必须保持门禁可回归。

- [Risk] 真实语义判定规则过严导致误报。  
  -> Mitigation: 采用“静态 + 运行证据”组合，并维护 allowlist/例外清单（最小化）。

- [Risk] README 同步门禁影响开发效率。  
  -> Mitigation: 提供统一 README 模板与自动检查提示，降低维护成本。

- [Risk] 交叉依赖导致与在途提案冲突。  
  -> Mitigation: a71 明确为独立提案，所有任务与产物仅在 `a71` 跟踪，禁止回写 a62 任务状态。

## Migration Plan

1. 建立 a71 模式替换矩阵（28 模式分批）。
2. 先实现 3 个基准模式（rag/structured-output/mcp）并固化门禁判定规则。
3. 批量替换剩余模式，按批次补齐 integration/replay 证据。
4. 全量更新 README/MATRIX/PLAYBOOK 的真实路径说明。
5. 接入并强制执行 a71 新门禁，完成 `quality-gate + docs-consistency` 全绿。

回滚策略：
- 若批次引入回归，按模式目录回滚该批次变更；
- 门禁脚本支持 `warn-only` 临时开关仅用于本地排查，不得进入主分支配置。

## Open Questions

- `real-runtime-semantic` 门禁中的“模式语义证据字段”是否采用统一前缀（如 `verification.semantic.*`）？
- 是否需要在 `tool/diagnosticsreplay` 为每个模式都新增独立 fixture，还是采用模式族共享 fixture？
- 是否引入“示例复杂度预算”防止 `main.go` 继续膨胀（如按 package 拆分）？
