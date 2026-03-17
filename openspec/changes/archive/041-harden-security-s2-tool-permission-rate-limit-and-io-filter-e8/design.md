## Context

S1 已完成依赖安全扫描与脱敏统一管线，但尚未把“工具执行权限 + 调用频率 + 模型 I/O 过滤”收敛到统一的运行时治理模型。当前 Action Gate 更偏交互确认与高风险策略，不等价于稳定可配置的安全策略层；同时 I/O 过滤缺少标准化扩展接口，难以在不同业务场景复用。

E8 目标是在 `pre-1.x` 阶段以最小可落地路径交付 S2：
- 工具安全策略在调用前可执行并可观测；
- 策略由 runtime config 驱动并支持热更新；
- 无效更新严格回滚；
- CI 具备独立阻断门禁。

约束：
- 保持 library-first，不引入对 CLI 的强依赖；
- 维持 Run/Stream 语义等价；
- 只做 `namespace+tool` 粒度与进程级限流，不扩展到多实例分布式计数；
- 违规行为采用 fail-fast `deny`，不引入告警后放行模式。

## Goals / Non-Goals

**Goals:**
- 新增工具权限治理：在 `namespace+tool` 粒度执行 allow/deny 决策。
- 新增工具频率限制：支持进程级配额窗口，超限即 `deny`。
- 新增模型输入/输出过滤扩展口：支持输入预过滤与输出后过滤链路。
- 在 runtime config 中落地 S2 配置并复用既有 `env > file > default` 与热更新机制。
- 补齐诊断字段与 reason code，支持审计权限拒绝、限流拒绝、过滤命中。
- 增加独立 CI 安全契约门禁，并可作为 required check。

**Non-Goals:**
- 不实现分布式限流计数（跨进程/跨节点共享配额）。
- 不在本阶段强制内置复杂 PII/注入检测规则库（保留扩展接口）。
- 不引入基于用户/会话维度的细粒度授权模型。
- 不改变现有 Action Gate 的交互确认语义与默认策略。

## Decisions

### Decision 1: 安全治理策略默认模式采用 `enforce`
- Choice: 对权限/限流/过滤策略默认直接生效，不提供默认 `dry_run`。
- Rationale: S2 目标是“可阻断”的安全闭环；仅观测不阻断无法满足高风险工具防护。
- Alternative considered: 默认 `dry_run`，后续手动切换。
- Rejected because: 在无强制阈值前提下，`dry_run` 更容易导致策略长期不落地。

### Decision 2: 权限与限流粒度固定为 `namespace+tool`
- Choice: 使用 `namespace + tool` 作为唯一匹配键，统一权限与限流策略索引。
- Rationale: 与现有 tool 路由边界一致，表达力与复杂度平衡较好，便于审计聚合。
- Alternative considered: 仅 `tool` 粒度或额外加入 `user/session` 粒度。
- Rejected because: 仅 `tool` 粒度隔离不足；`user/session` 超出 E8 变更域并提高实现复杂度。

### Decision 3: 限流计数作用域采用 `process`
- Choice: 限流计数在单进程内维护，窗口化统计命中后直接拒绝。
- Rationale: 与当前 runtime 部署形态和库内状态模型一致，实现与回滚语义简单可靠。
- Alternative considered: `global/distributed` 共享计数。
- Rejected because: 需要外部存储与一致性协议，不适合当前里程碑。

### Decision 4: 违规行为统一 `deny`（fail-fast）
- Choice: 权限拒绝、限流超限、过滤硬拒绝均走 fail-fast，阻断执行。
- Rationale: 安全策略应具有确定阻断能力，减少“命中但继续执行”的风险。
- Alternative considered: `warn` 或 `soft-block`。
- Rejected because: 会降低策略可信度，并增加后续行为歧义。

### Decision 5: I/O 过滤以可插拔接口优先，内置最小默认链路
- Choice: 为输入与输出分别提供过滤接口（前置/后置钩子），默认链路仅保证可装配与可观测。
- Rationale: 先建立契约与扩展点，后续可增量引入更强规则集，避免一次性重型策略。
- Alternative considered: 一次性内置完整规则引擎。
- Rejected because: 风险高、调参与误杀成本大，不利于快速收敛。

### Decision 6: 热更新遵循“成功原子切换，失败回滚”
- Choice: 安全配置更新必须通过完整校验后原子生效；失败时保留旧快照并输出 reload 错误诊断。
- Rationale: 避免策略半生效导致权限空窗或限流紊乱。
- Alternative considered: 部分字段容错生效。
- Rejected because: 会破坏配置一致性与审计确定性。

### Decision 7: 新增独立 `security-policy-gate` CI 门禁
- Choice: 将 S2 契约测试放入独立 job，作为 required check 候选。
- Rationale: 与现有 replay/template 门禁模式一致，便于分离责任与 branch protection 配置。
- Alternative considered: 合并到单一质量门禁脚本。
- Rejected because: 可见性与可治理性不足，难以独立追踪安全回归。

## Risks / Trade-offs

- [Risk] 进程级限流在多实例部署下无法提供全局配额一致性。
  - Mitigation: 明确 `process` 语义并预留未来外部计数器扩展接口。
- [Risk] I/O 过滤默认最小策略可能覆盖不足。
  - Mitigation: 提供清晰扩展接口与命中诊断字段，便于后续分阶段增强。
- [Risk] fail-fast deny 可能导致短期误拒绝影响可用性。
  - Mitigation: 强化配置校验与契约测试，提供可定位的 reason code。
- [Risk] 新增安全门禁延长 CI 耗时。
  - Mitigation: 将安全契约测试控制在最小闭环并与现有测试资产复用。

## Migration Plan

1. 扩展 runtime config schema，加入工具权限、限流、I/O 过滤配置字段与默认值。
2. 增加配置校验逻辑，覆盖枚举/范围/必填约束，并接入启动与热更新路径。
3. 在 tool dispatch 前接入权限与限流决策层，违规统一 `deny`。
4. 在 model 输入与输出链路接入 filter 扩展接口，补齐命中/拒绝诊断字段。
5. 扩展 diagnostics/event 映射，新增安全 reason code 与审计字段。
6. 新增安全契约测试并接入独立 CI `security-policy-gate` job。
7. 更新文档（配置字段索引、调试说明、CI required check 建议）。

## Open Questions

- 本阶段无阻断级开放问题；后续是否引入分布式限流与更强内置规则库留待下一里程碑评估。