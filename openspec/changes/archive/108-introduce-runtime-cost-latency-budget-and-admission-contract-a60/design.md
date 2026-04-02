## Context

A58 正在收敛跨策略优先级与解释链，A59 正在收敛 memory scope/search/lifecycle 合同。下一步运行时缺口在于“预算 admission”尚未形成统一契约：
- token/tool/sandbox/memory 开销分散在多个子域，缺少统一预算快照；
- admission 当前主要基于 readiness/策略阻断，未冻结预算驱动的 `allow|degrade|deny` 规则；
- 在高负载与跨 provider 场景下，缺少可回放的预算决策输出，导致回归不可稳定拦截。

A60 的目标是在 `library-first + contract-first` 下补齐预算 admission 合同，不引入平台控制面，不改写 A58/A59 既有语义。

## Goals / Non-Goals

**Goals:**
- 冻结统一预算快照：`cost + latency` 覆盖 token/tool/sandbox/memory。
- 冻结 admission 决策：`allow|degrade|deny` 的 deterministic 判定规则。
- 冻结降级动作：`degrade_action` 可配置、可观测、可回放。
- 将预算 admission 输出接入 QueryRuns additive 字段、replay fixture 与独立 gate。
- 保持 Run/Stream 准入语义等价和 deny path side-effect-free。

**Non-Goals:**
- 不引入托管预算控制面、远程限流服务或平台化调度中心。
- 不重定义 A58 的 `policy_decision_path/deny_source` 或 A59 的 memory 生命周期语义。
- 不在 A60 内展开性能调优（A64）或示例收口（A62）。
- 不在 A60 之外再拆分“预算 admission”同域提案；后续需求仅作为 A60 增量任务吸收。

## Decisions

### Decision 1: 预算模型采用统一快照对象

- 方案：在 admission 前计算统一 `budget_snapshot`，包含至少 `cost_estimate` 与 `latency_estimate`，并带来源分解（token/tool/sandbox/memory）。
- 备选：各模块单独做 budget 判断。
- 取舍：单快照更利于回放与解释，避免入口与子模块漂移。

### Decision 2: 准入判定采用两阶段规则

- 方案：先做硬阈值判定（超阈值即 `deny`），再做降级阈值判定（进入 `degrade`），其余为 `allow`。
- 备选：单阈值 + best-effort。
- 取舍：两阶段更贴合“可运行但降级”和“必须阻断”的现实需求，且便于明确 drift 分类。

### Decision 3: 降级动作采用策略驱动且输出显式 action

- 方案：通过 `runtime.admission.degrade_policy.*` 管理降级动作集合与顺序，至少输出一个 canonical `degrade_action`。
- 备选：降级动作内嵌在业务分支，不做统一字段。
- 取舍：统一 action 字段可被 replay/gate 稳定断言，降低联调不透明度。

### Decision 4: 预算 admission 与 A58/A59 输出做“引用而不重定义”

- 方案：A60 在解释链中复用 A58 policy winner 与 A59 memory 指标作为预算输入来源，不重定义同义字段。
- 备选：复制一套 budget 专用来源字段。
- 取舍：避免语义分叉，保持跨提案字段一致性。

### Decision 5: 回放与门禁与 contract 同步冻结

- 方案：新增 `budget_admission.v1` fixture 与 drift taxonomy，并通过 `check-runtime-budget-admission-contract.*` 独立阻断。
- 备选：仅在集成测试验证。
- 取舍：独立 gate 更适合持续拦截阈值/判定漂移，避免回归潜伏到主线。

### Decision 6: 增加同域收口护栏，禁止平行语义分叉

- 方案：在 A60 合同中显式增加两类护栏：
  - `control_plane_absent`：预算 admission 仅允许库内嵌入式执行，不演进为托管控制面；
  - `field_reuse_required`：必须复用 A58/A59 既有解释字段，不得新增同义预算来源字段。
- 备选：仅在 proposal 层口头约束。
- 取舍：口头约束无法被 gate 与回放稳定验证，容易再次拆出同域提案。

## Risks / Trade-offs

- [Risk] 预算估算偏差导致过度 deny 或过度 degrade  
  → Mitigation: 阈值分层（hard/degrade）+ replay fixture 持续校准 + 阈值文档化。

- [Risk] 新增预算计算路径带来 admission 前开销  
  → Mitigation: 预算快照计算保持有界复杂度，并在 A64 前不引入重型优化。

- [Risk] 与 A58/A59 并行联调时字段解释冲突  
  → Mitigation: A60 明确“引用现有字段，不重定义同义字段”，并用 index + gate 固化。

- [Risk] 降级动作过多造成行为不可预测  
  → Mitigation: canonical degrade action 白名单 + deterministic 顺序执行。

- [Risk] 后续需求以“补丁提案”方式重复进入同域  
  → Mitigation: 将同域增量纳入 A60 任务编排与 gate 断言，不新增平行预算 admission 提案。

## Migration Plan

1. 扩展 `runtime/config`：新增 budget 与 degrade policy 字段、默认值与 fail-fast 校验。
2. 在 admission guard 接入预算快照构建与两阶段判定规则。
3. 在 diagnostics/recorder 增加 `budget_snapshot`、`budget_decision`、`degrade_action` additive 字段。
4. 在 replay tooling 增加 `budget_admission.v1` 及 drift 分类。
5. 新增并接入 `check-runtime-budget-admission-contract.sh/.ps1` 到质量门禁。
6. 同步文档与主线 contract index，确保 required-check 映射一致。

回滚策略：
- 热更新失败自动回滚到上一个有效快照；
- 可通过关闭 budget admission 新字段恢复到现有 readiness/admission 基线行为。

## Open Questions

- None. A60 按 roadmap 验收口径一次性冻结预算 admission contract，不再拆同域平行提案。
