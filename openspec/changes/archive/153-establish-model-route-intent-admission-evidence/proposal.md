## Why

归档的 `model-catalog-routing-admission-audit` 已证明 Baymax 能对 host-supplied 候选执行能力、credential、fallback、readiness 与确定性 admission，但尚未定义“宿主明确要求某个路由意图”如何与这些既有事实逐项比对。没有这一层证据，继续扩展本地路由容易把候选排序、发现或全局状态误当成产品需求。

本 change 先建立离线、只读、可回放的 route-intent admission evidence 合同，用来回答现有 catalog/admission 事实是否满足显式路由意图；只有证据明确显示存在缺口，后续 change 才能讨论运行时路由行为。

## What Changes

- 新增版本化、provider-neutral 的 route-intent evidence 输入与归一化输出。
- 将宿主提供的目标身份、允许候选、必需/可选能力、credential/readiness 事实与现有 catalog admission 结果进行确定性比对。
- 固定三类顶层结论：`satisfied`、`route-gap-confirmed`、`insufficient-evidence`，并提供有界的字段级 reason taxonomy。
- 增加离线 replay fixture、漂移检测、幂等性、隐私边界、Run/Stream parity 和 contribution gate 覆盖。
- 保持现有 exact-identity admission、catalog resolver、fallback、readiness 及 catalog generation 语义不变；不新增运行时路由器或配置控制面。

## Example Impact Assessment

无需示例变更（附理由）：本 change 仅增加离线证据、fixture、replay 与 gate，不改变 `examples/agent-modes` 的配置、runtime path、expected markers 或可观察行为。若后续 change 引入实际路由选择，必须先更新 `MATRIX.md` 与受影响模式 README，并重新评估示例影响。

## Capabilities

### New Capabilities

- `model-route-intent-admission-evidence`: 定义显式宿主路由意图与既有模型 catalog/admission 事实之间的有界、可回放比较合同。

### Modified Capabilities

无。现有 `model-catalog-routing-admission-audit`、`provider-model-capability-and-credential-preflight` 与 readiness 合同继续作为事实来源，本 change 不改变其要求。

## Impact

- 受影响范围：`model/catalog` 的 provider-neutral projection 接缝、`tool/diagnosticsreplay`、`tool/contributioncheck`、fixture/gate 与相关文档索引。
- 不新增 provider SDK 依赖、remote discovery、background refresh、credential store/probe、全局 mutable router、运行时配置键或第二套 readiness/terminal 状态机。
- 证据仅保存规范化身份、枚举、摘要、长度、generation、bounded reason 与 parity 结果；禁止 endpoint、credential material、raw provider payload、prompt、reasoning 或无界 body。
- 回滚点：删除本 change 的 evidence projection、fixture、replay 与 gate 即可；不涉及持久化迁移或远端状态清理。
