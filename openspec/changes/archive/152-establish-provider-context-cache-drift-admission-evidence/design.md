## Context

见 [proposal.md](proposal.md) 的动机。现有 `provider_request_projection.v1` 已固定 supported Provider 的 canonical source/observed facts、role/tool-result/order drift、Run/Stream parity 与 cache usage 语义；归档 142、144、148 已分别完成审计、原生投影修复和 cache usage carrier。新 change 必须保留这些 artifact 的版本和 digest，而不是把新的宿主或 SDK 观察塞入既有请求投影 fixture。

## Goals / Non-Goals

**Goals:**

- 以 `provider_context_cache_evidence.v1` 建立一个有界、SDK-neutral、reference-only 的准入证据包。
- 将 evidence package 的 projection reference 与既有 `provider_request_projection.v1` case/digest 逐一验证，并对结构化上下文、cache parity 和可选宿主成本摘要给出可重放 verdict。
- 在 `no-drift`、`drift-confirmed`、`insufficient-evidence` 之间保持穷尽、互斥、确定性的语义，并使 shell/PowerShell gate 对等。
- 将任何 confirmed drift 限制为 review-only、owner-routed 的后续修复输入。

**Non-Goals:**

- 不修改 `ModelRequest`、adapter 的 native request constructor、`ModelResponse.CacheUsage`、`TokenUsage`、`RuntimeRecorder` 或既有 projection fixture schema/digest。
- 不采集 live provider、凭证、网络数据、raw SDK payload、逐请求成本、价格表、Prompt 或 reasoning。
- 不创建自动修复、自动 cache 策略、router、credential store、runtime 配置或第二套 Provider wire protocol。
- 不借本 change 新增或永久豁免 Go 文件行数例外；仅允许对已有 A63 staged-split 例外做一次明确、短期、可审计的到期续期。

## Decisions

### 1. 新建引用式 evidence contract，而不扩展 `provider_request_projection.v1`

每条 evidence 持有 provider、SDK identity、projection case identity、canonical digest、声明维度和结论所需的有界摘要。replay 先解析既有 projection evidence，再校验引用，最后归一化 verdict。

**Rationale:** 保持归档 142/144/148 已冻结的 request-projection 语义、版本和历史 digest；evidence 的生命周期和宿主来源可独立演进。

**Alternatives considered:**

- 将 SDK/成本字段加入 `provider_request_projection.v1`：会改变既有 fixture 的职责与 digest，并把宿主证据误建模成 runtime request facts，拒绝。
- 只写文档检查表：无法 replay、无法证实 reference 关联，也无法跨 shell gate，拒绝。

### 2. 采用三态 verdict 与显式 required dimensions

fixture 明确哪些维度是本次准入所必需；每个维度只可产生 verified-no-drift、verified-drift 或 unavailable。最终 verdict 按以下优先级导出：verified drift 优先于证据不足；若无 drift 且有任何 required unavailable 则为 evidence insufficient；其余才为 no drift。

**Rationale:** 将“没有数据”与“数据证明正常”严格分开，并且在证据同时不完整又发现真实漂移时保留风险更高的结论。

**Alternatives considered:**

- 二态 pass/fail：把缺失证据混入 pass，拒绝。
- 由 host 任意文本给 verdict：不可审计、不可复放，拒绝。

### 3. 性能只保存有界、单位显式的摘要

成本/P95 evidence 包含 metric identity、unit、样本数、baseline/observed P50/P95、阈值与 projection digest。验证器校验其数学关系和阈值，不读取原始时序或价格信息。

**Rationale:** 允许宿主提供稳定成本/延迟退化证据，同时不把 host 定价、用户行为或运行日志写入仓库。

**Alternatives considered:**

- 读取 live benchmark 或 provider billing API：引入网络、凭证、可变成本与不可重放行为，拒绝。
- 从普通 token usage 推导成本：违背现有 cache usage contract，拒绝。

### 4. Gate 只验证与路由，不修复

`tool/diagnosticsreplay` 提供纯函数 parse/validate/replay，`tool/contributioncheck` 只汇总 verdict、digest 和 reason codes。Gate 遇到 `drift-confirmed` 返回可操作但 review-only 的 owner route；遇到 malformed evidence 则 fail-fast。gate 不修改 fixture、adapter 或 docs。

**Rationale:** 保持 diagnostics replay 的离线、单向证据职责，避免将质量 gate 变成运行时决策或自修改控制面。

### 5. 对既有 line-budget 例外执行窄范围续期

本 change 的实现不触碰业务 Go 文件，但完整 quality gate 被 11 条已有 A63 staged-split 例外的到期日阻断。续期仅更新这些既有路径的 `expiry` 为 `2026-12-31`，保留原 owner、reason、baseline_lines 与 `allow_growth`；不新增例外、不提高阈值、不允许原本禁止增长的文件增长。到期前必须由治理维护者拆分文件或重新完成例外审查。

**Rationale:** 让本 change 的验证恢复可执行，同时把治理债务显式限定在已有清单和固定期限内；不把质量门禁失败静默转化为通过。

## Risks / Trade-offs

- [宿主只提供不完整性能信息] → 明确输出 `insufficient-evidence`，不默认通过，也不阻塞无关 Provider 行为。
- [治理例外续期掩盖代码膨胀] → 只更新 11 条已有 A63 staged-split 路径的到期日，保留 `allow_growth` 与 baseline，设定 2026-12-31 复审点，并在 tasks 中记录回滚/拆分责任。
- [SDK identity 变化但没有实际语义变化] → identity 是可追溯 metadata，不单独触发 drift；只有已验证维度矛盾才确认 drift。
- [新的 evidence contract 与既有 projection contract 脱节] → 强制 case identity 与 canonical digest reference，并覆盖 reference mismatch 测试。
- [统计摘要被伪造或口径混用] → 只接受单位显式、样本受限、P50/P95 有序且阈值可重算的 evidence；其余 fail-fast。
- [gate 结论被误解为自动修复授权] → verdict 附带固定的 review-only route，tasks/spec 明确禁止自动改动 runtime 或 fixture expectation。

## Migration Plan

1. 添加纯离线 evidence model、fixture parser/validator 与 canonical replay，并保留既有 request-projection fixture 和 parser 原样。
2. 添加三家 Provider 的 seed evidence，以及 malformed、privacy、bounds、parity、成本与 historical compatibility cases。
3. 接入 shell/PowerShell contribution gate、test index、模块文档和 Roadmap；先以 evidence-only 结果运行。
4. 若 gate 后续给出 `drift-confirmed`，单独从最新 `master` 发起一个 owner-scoped OpenSpec repair change；本 change 不承载该修复。

**Rollback:** 移除新 evidence fixture/replay/gate 及其文档映射；既有 `provider_request_projection.v1`、adapter、响应 carrier、配置和持久化数据无需迁移或回滚。

## Example Impact Assessment

无需示例变更（附理由）：实现只增加离线 SDK/宿主证据的 fixture、replay 和 gate。它不改变 `examples/agent-modes` 的配置或可观察运行时语义；若后续独立 repair change 影响示例 markers，必须由该 repair change 先完成示例文档基线。
