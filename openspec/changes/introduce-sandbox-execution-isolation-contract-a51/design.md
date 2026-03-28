## Context

Baymax 当前已具备 S2（permission/rate-limit/io-filter）与 S3/S4（deny-only 安全事件及投递治理）能力，但工具执行层仍缺少“执行隔离”标准接缝。  
在不引入平台化控制面的前提下，需提供可嵌入、可替换的沙箱执行 SPI，并与既有 readiness/admission/diagnostics/replay 体系保持一致。

## Goals / Non-Goals

**Goals:**
- 提供宿主注入式 sandbox execution SPI，支持 `host|sandbox|deny` 决策结果。
- 冻结跨后端通用的 ExecSpec/ExecResult 语义，确保 nsjail/bubblewrap/docker/gVisor/Windows Job 可映射。
- 固化 executor capability negotiation（probe + required capabilities）并接入 fail-fast 准入链路。
- 将 sandbox 配置纳入统一配置治理（`env > file > default`、fail-fast、热更新回滚）。
- 在 Run/Stream、readiness/admission、diagnostics/replay 路径固化一致语义。
- 一次性补齐 sandbox 可观测性闭环（timeline、security event、delivery、single-writer、replay、gate）。
- 提供 `observe|enforce` 渐进启用策略与 `required` 硬约束机制。

**Non-Goals:**
- 不内置 Docker/K8s/VM 控制平面，不引入平台化运维能力。
- 不承诺跨主机或跨内核强隔离等级，隔离强度由宿主执行器实现决定。
- 不改变现有 provider fallback、scheduler/workflow/a2a 的核心终态契约。

## Decisions

### Decision 1: 采用宿主注入式 SandboxExecutor SPI，避免运行时内置平台依赖

- 方案：
  - 在 `core/types` 定义 sandbox 执行接口与执行结果模型（含 timeout/exit-code/violation 语义）。
  - 在 `runner.New(...Option)` 注入 executor；未注入时按策略处理（fallback 或 deny）。
- 取舍：
  - 相比内置容器编排实现，SPI 方案更符合 lib-first 边界，且跨平台可替换。

### Decision 2: 固化三态动作 `host|sandbox|deny` 与双模式 `observe|enforce`

- 方案：
  - policy resolve 输出动作：`host`（直接执行）、`sandbox`（隔离执行）、`deny`（fail-fast）。
  - `observe` 模式记录“本应 sandbox/deny”决策，但不改变执行主路径；`enforce` 模式严格执行动作。
- 取舍：
  - 提供灰度迁移路径，降低一次性切换风险；代价是状态空间更大，需要更强契约测试覆盖。

### Decision 3: 将 sandbox.required 纳入 readiness/admission 强约束

- 方案：
  - `sandbox.required=true` 时，若 executor 不可用或 profile 非法，preflight 产出 blocking finding。
  - admission 在 managed Run/Stream 前置 deny，保持 side-effect free。
- 取舍：
  - 与 A44 一致的 fail-fast 准入能降低误执行风险；但会提高启用门槛。

### Decision 4: 诊断与回放新增 sandbox additive 字段与 drift 分类

- 方案：
  - run diagnostics 增加 `sandbox_*` 字段，并保持 `additive + nullable + default`。
  - replay tooling 增加 sandbox fixture（建议 `a51.v1`）与 drift 分类：
    - `sandbox_policy_drift`
    - `sandbox_fallback_drift`
    - `sandbox_timeout_drift`
    - `sandbox_capability_drift`
    - `sandbox_resource_policy_drift`
    - `sandbox_session_lifecycle_drift`
- 取舍：
  - 增量字段和 fixture 会增加维护成本，但可显著提升可回归性与排障效率。

### Decision 5: 门禁新增独立 sandbox contract check 并接入质量总门

- 方案：
  - 新增 `check-security-sandbox-contract.sh/.ps1`。
  - 在 `check-quality-gate.*` 接入并保持 shell/PowerShell parity。
- 取舍：
  - 增加 CI 时长，但确保 sandbox 语义退化可被阻断。

### Decision 6: Sandbox observability contract一次性覆盖四个层面

- 方案：
  - Timeline：新增 sandbox canonical reasons（policy deny / launch failed / fallback / timeout）。
  - Security Event：sandbox deny 纳入 S3 taxonomy，并复用 S4 deny-only delivery。
  - Diagnostics：新增 `sandbox_*` additive 字段并复用 single-writer + idempotency 规则。
  - Replay/Gate：新增 `a51.v1` sandbox fixture + drift 分类，并接入独立 gate。
- 取舍：
  - 一次性范围更大，但可避免后续“补 observability 契约”的二次提案与语义分裂。

### Decision 7: 冻结 ExecSpec/ExecResult 作为跨后端最小公共面

- 方案：
  - `ExecSpec` 最小字段冻结为：`command/args/env/workdir/mounts/network/resource_limits/session_mode/timeouts`。
  - `ExecResult` 最小字段冻结为：`exit_code/stdout/stderr/timed_out/oom_killed/violation_codes/resource_usage`。
  - 工具与 MCP 路径统一通过该模型进入 sandbox executor。
- 取舍：
  - 首期字段更多，但避免后续因后端差异重复扩展提案。

### Decision 8: 引入 capability negotiation，避免“配置可写但后端不支持”

- 方案：
  - executor 提供 capability probe 输出（如 `network_off/readonly_root/pid_limit/cpu_limit/memory_limit/mount_rw_allowlist/session_per_call/session_per_session`）。
  - `security.sandbox.required_capabilities` 声明必需能力。
  - 在 `enforce` + `required` 场景下，能力不满足即 blocking deny。
- 取舍：
  - 增加启动前校验复杂度，但显著降低线上运行期意外降级。

### Decision 9: 工具形态桥接采用“双通道”：in-process 与 process-adapter

- 方案：
  - 保持现有 `Tool.Invoke` 兼容。
  - 新增可选 adapter 接口用于构建 `ExecSpec`（适配 shell/file/process 工具）。
  - `sandbox` 动作命中但工具未适配时：
    - `observe`: host 执行 + 记录 `sandbox_tool_not_adapted`
    - `enforce`: deny（默认）或按显式 fallback 执行
- 取舍：
  - 迁移成本可控，不强迫所有历史工具一次性改造。

### Decision 10: MCP stdio 沙箱会话语义首期冻结为 `per_call|per_session`

- 方案：
  - `session_mode=per_call`：每次调用创建独立 sandbox 执行单元。
  - `session_mode=per_session`：client session 绑定一个 sandboxed transport 生命周期。
  - reconnect/close 行为在两种模式下保持 deterministic 语义。
- 取舍：
  - 明确生命周期契约，降低不同后端实现行为漂移。

### Decision 11: A51 首期同时交付 sandbox executor conformance harness

- 方案：
  - 在 A51 内新增 sandbox executor conformance harness suites，覆盖 canonical ExecSpec/ExecResult、capability negotiation、session lifecycle、fallback 语义。
  - harness 采用 offline deterministic fixtures，并接入 sandbox quality gate。
- 取舍：
  - 增加首期实施工作量，但可避免后续再拆 sandbox harness 子提案。

## Risks / Trade-offs

- [Risk] sandbox 策略误配导致误拒绝或行为突变。  
  -> Mitigation: 默认 `enabled=false`，上线先 `mode=observe`，并提供 `fallback_action`。

- [Risk] 隔离执行器启动开销带来尾延迟上升。  
  -> Mitigation: 引入 profile 分级与 `max_concurrency`，优先对高风险工具启用。

- [Risk] 跨平台执行器行为不一致（Linux/Windows/macOS）。  
  -> Mitigation: 接口语义冻结 + 契约测试矩阵，仅要求标准化结果字段，不强制底层实现一致。

- [Risk] sandbox 失败回退路径造成安全语义漂移。  
  -> Mitigation: `fallback_action` 枚举固定且可观测，默认建议 `deny` 于生产环境。

- [Risk] 能力探测与实际执行能力不一致（虚假 capability）。  
  -> Mitigation: capability probe + fixture 回放 + 跨后端矩阵集成测试，发现漂移即 gate 阻断。

- [Risk] 运行时字段扩展导致 diagnostics 查询成本上升。  
  -> Mitigation: sandbox 字段纳入 A45 cardinality 治理和 A42 benchmark 回归门禁。

## Migration Plan

1. 新增 `security.sandbox.*` 配置结构、默认值、校验与热更新回滚测试。
2. 在 `core/types` 增加 `ExecSpec/ExecResult/SandboxExecutor` 与 capability probe 契约。
3. 在 `core/runner` + `tool/local` + `mcp/stdio` 接入 sandbox 决策、工具桥接与会话模式分支。
4. 在 `runtime/config/readiness*` 与 admission guard 增加 required capabilities 预检与阻断语义。
5. 在 `runtime/diagnostics` 与 `RuntimeRecorder` 增加 backend/capability/resource/latency 类 sandbox 字段与计数聚合。
6. 在 timeline/security-event/security-delivery 路径补齐 sandbox reason/taxonomy/delivery 语义。
7. 在 `tool/diagnosticsreplay` 与 `integration` 增加 A51 fixture/drift/parity/idempotency/capability-matrix 套件。
8. 在 `scripts/check-quality-gate.*` 接入 sandbox gate 与跨后端 smoke suite，并同步文档索引。

## Open Questions

- None for A51 scope. 本提案已冻结 sandbox 接入与观测的最小完整 contract，后续仅做实现扩展与后端适配增量。
