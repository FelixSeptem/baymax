## Context

当前主线已经具备：
- A56：ReAct Run/Stream parity 与终止 taxonomy；
- A58：跨域决策解释链；
- A67：plan notebook + plan-change hook 合同；
- A68：realtime event protocol（实施中）。

但 ReAct/JIT context 组织仍存在结构性缺口：引用与正文混合注入、子代理回传结构不稳定、编辑动作缺少收益门槛、swap-back 粒度粗、lifecycle 分层不统一。结果是等价请求在不同入口下容易出现上下文噪声差异和回放漂移。roadmap 明确 A67-CTX 是 A68 后下一顺位，因此本设计聚焦“合同层收口”，在不改变既有 ReAct 主语义的前提下收敛 context 组织治理。

## Goals / Non-Goals

**Goals:**
- 定义 reference-first 两段式注入合同与校验边界。
- 定义 isolate handoff payload 合同与主/子代理消费约束。
- 定义 edit gate 的 `clear_at_least` 阈值策略与判定可观测性。
- 定义 relevance swap-back 与 `hot|warm|cold` lifecycle tiering 合同。
- 定义 task-aware recap 结构化输出合同。
- 增加 `runtime.context.jit.*` 配置治理、A67-CTX additive 诊断字段、fixture/gate 阻断。
- 保证 Run/Stream 等价与 A56/A58/A57 边界不漂移。

**Non-Goals:**
- 不引入平行 ReAct loop、平行决策链或新终止 taxonomy。
- 不引入托管 context 控制面、远程上下文路由服务或平台化会话管理。
- 不把 A67-CTX 扩展为性能专项；性能热点统一归 A64。
- 不改写 provider 接入方式；`context/*` 继续通过既有抽象层访问模型能力。

## Decisions

### Decision 1: 采用 reference-first 两段式注入，默认“先引用后正文”

- 方案：
  - stage2 先执行 `discover_refs` 输出候选引用；
  - 再执行 `resolve_selected_refs` 对被选引用按预算展开正文。
- 备选：维持单段式直接注入全文。
- 取舍：两段式可在 token 预算下稳定控制噪声，同时提升回放可解释性。

### Decision 2: isolate handoff 固化结构化 payload，主代理默认只消费摘要+证据引用

- 方案：子代理回传固定字段 `summary/artifacts/evidence_refs/confidence/ttl`，正文内容仅在满足策略时展开。
- 备选：允许子代理回传自由文本并直接拼接到主上下文。
- 取舍：固定结构可避免子代理输出污染主上下文，并支持 deterministic replay。

### Decision 3: context edit gate 采用收益阈值守门，不达标不清理

- 方案：引入 `clear_at_least` 与收益比（estimated saved tokens / stability cost）阈值；未达阈值仅记录建议，不执行激进编辑。
- 备选：始终按固定规则清理上下文。
- 取舍：阈值守门可避免“为清理而清理”导致信息损失和行为波动。

### Decision 4: swap-back 从 run 粒度升级为 query+evidence 相关性回填

- 方案：回填依据当前 query 与 evidence tags 计算相关性分数，达到阈值才回填。
- 备选：按 run 末尾统一回填。
- 取舍：相关性回填减少无关上下文回灌，并提升当前轮问题命中率。

### Decision 5: lifecycle tiering 固定 `hot|warm|cold` 与 TTL/淘汰策略

- 方案：统一 write/compress/prune/spill 的分层动作，所有跨层迁移记录 canonical reason。
- 备选：各模块自行维护本地缓存淘汰逻辑。
- 取舍：统一分层是稳定预算与回放断言的前提。

### Decision 6: recap 固定 task-aware 结构化来源标签，不再用固定模板兜底

- 方案：recap 输出必须带 `context_recap_source`，并反映本轮真实选择、剪裁和外化动作。
- 备选：沿用固定模板 recap。
- 取舍：task-aware recap 可显著提升解释链可读性，并降低“模板化但不相关”噪声。

### Decision 7: fixture-first + 独立 gate + 边界断言

- 方案：新增 5 个 A67-CTX fixture 与 6 类 drift taxonomy；新增 `check-context-jit-organization-contract.*` 并接入质量门禁；增加 `context_provider_sdk_absent` 边界断言。
- 备选：只补集成测试，不设专项 gate。
- 取舍：专项 gate 对上下文语义漂移拦截更稳定，也能持续约束架构边界。

## Risks / Trade-offs

- [Risk] 两段式注入增加流程复杂度，可能影响已有 context 路径。
  -> Mitigation: 默认保持 feature-gated；先新增 contract tests，再接入主路径。

- [Risk] edit gate 阈值配置不当，导致清理过度或清理不足。
  -> Mitigation: 配置 fail-fast + 边界测试 + 诊断字段输出实际 gate decision。

- [Risk] relevance score 不稳定导致 swap-back 抖动。
  -> Mitigation: 固定 score 计算输入与阈值含义，增加 replay drift 分类 `swapback_relevance_drift`。

- [Risk] lifecycle 分层可能与 A68 interrupt/resume 恢复边界冲突。
  -> Mitigation: 规定 A68 cursor 边界优先；A67-CTX 仅在合法恢复边界内执行层级迁移。

- [Risk] 新增字段扩大 QueryRuns 负载。
  -> Mitigation: 字段全部 additive + nullable，保持单写入口与解析兼容测试。

## Migration Plan

1. 配置层：在 `runtime/config` 增加 `runtime.context.jit.reference_first.*`、`isolate_handoff.*`、`edit_gate.*`、`swap_back.*`、`lifecycle_tiering.*`，并实现启动校验与热更新原子回滚。
2. 注入层：在 `context/*` 接入 `discover_refs -> resolve_selected_refs` 两段式流程，保持关闭开关时行为不变。
3. handoff 层：定义子代理回传 payload schema，接入主代理消费策略与 TTL 处理。
4. gate 层：实现 `clear_at_least` 判定与执行分流（执行/拒绝），并回写 gate decision 诊断。
5. swap-back 与 tiering 层：引入 query+evidence 相关性回填与 `hot|warm|cold` 生命周期策略。
6. recap 层：将 tail recap 升级为 task-aware 结构化 recap，并标记 `context_recap_source`。
7. 可观测层：在 `runtime/diagnostics` 与 `RuntimeRecorder` 增加 A67-CTX additive 字段，保持单写幂等。
8. 回放层：新增 5 个 fixtures、6 类 drift 分类、mixed fixtures 兼容测试。
9. 门禁层：新增 `check-context-jit-organization-contract.sh/.ps1`，并接入 `check-quality-gate.*` 与 impacted-contract suites。
10. 文档层：同步 runtime config/diagnostics、contract index、roadmap 与 README。

回滚策略：
- 配置回滚：热更新非法配置自动回滚到上一个有效快照。
- 功能回滚：关闭 `runtime.context.jit.*.enabled` 相关开关即可恢复旧路径。
- 数据兼容：新增诊断字段保持 additive，旧消费者可忽略。

## Open Questions

- None. A67-CTX 按 roadmap 一次性收口 context 组织同域需求，不再拆分平行提案。
