## Context

现有 `model/catalog` 已拥有 host-supplied catalog、candidate identity、capability/credential/readiness admission 与条件化 resolver；`tool/diagnosticsreplay` 已拥有离线 fixture 和 drift 校验能力。当前缺口是缺少一个把“宿主路由意图”与这些既有事实进行显式比较的合同。设计必须保持 library-first 分层、`env > file > default`、fail-fast + atomic rollback、additive nullable/default diagnostics 兼容，以及 Run/Stream 语义对等。

## Goals / Non-Goals

**Goals:**

- 定义 `model_route_intent_admission.v1` 的 bounded input、canonical output、verdict 与 reason taxonomy。
- 复用既有 catalog/admission facts，形成纯函数、离线、reference-only 的比较路径。
- 用 fixture/replay/gate 证明满足、缺口已证实、证据不足、漂移、隐私违规和 parity 语义。
- 让后续运行时路由提案以可审计证据为前提，而非在本 change 中引入行为。

**Non-Goals:**

- 不实现新的 runtime router、自动模型切换、remote discovery、background refresh 或配置控制面。
- 不存储或探测 credential，不引入 endpoint/raw provider data，也不改变 resolver ranking。
- 不改变现有 exact-identity admission、fallback、readiness、terminal semantics 或 examples/agent-modes。

## Decisions

### 1. Evidence projection stays reference-only

比较输入引用 host-supplied intent 与已存在的 catalog/admission facts；输出只保留 normalized identity、generation、enum、digest/length、bounded reasons 与 parity。这样可以复用已有 owner，避免复制事实源或产生第二本路由账本。

备选方案：把意图直接接入 runtime selector。拒绝，因为会在缺少 gap evidence 时冻结新的运行时行为。

### 2. Three verdicts, no implicit selection

`satisfied` 表示现有合同已能证明要求；`route-gap-confirmed` 表示有明确、可复现的表达缺口；`insufficient-evidence` 表示事实缺失或矛盾。比较器永远不按 caller order 选模，也不以“无法判断”冒充缺口。

备选方案：直接返回 boolean 或自动触发 resolver。拒绝，因为无法区分证据不足与产品缺口，且会越过提案边界。

### 3. Replay-first contract and taxonomy

使用独立的 versioned fixture 与 diagnostics replay handler，校验 canonical digest、reason order、generation、privacy、bounds 和 Run/Stream parity。未知字段忽略、缺失可选字段使用默认值，保证历史 fixture 兼容。

备选方案：把证据写入 runtime event stream。首阶段拒绝；若后续需要诊断字段，必须通过 `RuntimeRecorder` 以 additive nullable/default 方式另行设计。

### 4. Boundary gates are part of the contract

contributioncheck 与 PowerShell/shell gate 检查不引入 provider SDK、network、credential store、global mutable router、raw payload 或第二 admission owner。任何越界在合并前失败。

## Risks / Trade-offs

- **[Risk] 意图字段过于抽象，始终返回 `insufficient-evidence`。** → 先固定最小可验证字段集和正/负/边界 fixture，并把新增字段作为版本化变更。
- **[Risk] reason taxonomy 与既有 catalog taxonomy 漂移。** → 复用现有稳定 reason，新增分类集中维护并由 replay/contribution gate 对等校验。
- **[Risk] evidence 被误当成运行时路由授权。** → 输出明确声明 reference-only，禁止 selected identity 产生新的 runtime side effect；文档与 gate 同时检查。
- **[Risk] 大型候选集导致隐私或资源压力。** → 对身份、候选、reason、digest 和序列化字节设置硬上界，超界 fail-fast。

## Migration Plan

1. 先实现 schema、normalization 与纯比较测试，再加入 canonical fixtures。
2. 接入 replay、contributioncheck、docs/gate，验证历史 fixture 与重复回放稳定。
3. 仅在 evidence 明确返回 `route-gap-confirmed` 时，将结果作为后续 runtime-routing change 的输入；本 change 不改变生产选择。
4. 回滚时移除 projection、fixtures、replay handler 与 gates；既有 catalog/admission 代码和数据无需迁移。
