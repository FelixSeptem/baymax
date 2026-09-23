## Example Impact Assessment

无需示例变更（附理由）：首阶段仅增加离线 schema pressure、selection quality、fixture、replay、benchmark 与 gate，不改变 `examples/agent-modes` 的 runtime path、配置键、工具准入或 expected markers。若后续运行时按需投影进入示例上下文，必须另行完成文档基线并重新评估该声明。

## Why

已准入的本地工具、MCP 工具和扩展工具都能提供 JSON schema，但当前没有统一、可回放的证据来回答两个问题：完整工具集合是否已经造成稳定的 schema/context 压力，以及缩小候选集合是否真的改善工具选择质量。没有这两类证据就直接引入 schema-on-demand，容易把宿主准入、安全边界和运行时请求语义混在一起。

本 change 先建立 audit-first 基线：用强制的合成 task fixture 测量压力和选择质量，用可选的 Eval corpus/Badcase 提供只读 advisory evidence，并离线比较多个确定性策略。只有证据同时证明存在稳定压力、质量退化和可重复的策略改善时，才把结果标记为后续 projection candidate；本 change 不改变运行时行为。

## What Changes

- 新增版本化、provider-neutral 的 `tool_schema_pressure_selection_audit.v1` 契约，描述已准入工具快照、bounded schema facts、task gold set、压力预算、质量指标、策略参数和 audit conclusion。
- 增加 canonical schema normalization、digest、byte/token estimate、tool-count pressure、selection precision/recall/F1、expected/forbidden hit 和 evidence sufficiency 规则。
- 增加多个离线 deterministic strategy scoring 轨道：完整 admitted set、capability filter、priority top-k、source partition，以及受版本化参数约束的 fixture-declared strategy；评分结果只用于 advisory，不生成运行时 tool subset。
- 增加 synthetic fixture 强制 gate 与 Eval corpus/Badcase 可选 advisory 双轨；corpus 缺失、未知字段或覆盖不足不得使强制 gate 失败。
- 增加 replay、benchmark、Run/Stream parity、历史默认值、unknown field、overflow、gold-set 冲突和 strategy drift 覆盖，并提供 shell/PowerShell gate。
- 增加边界守卫，确保不引入动态下载、marketplace、credential store、provider tokenizer、全局 selector/router、第二准入/状态机或对 `ModelRequest`/Provider projection 的运行时接线。

## Capabilities

### New Capabilities

- `tool-schema-pressure-selection-audit`: 对已准入工具集合执行有界、离线、可回放的 schema pressure 与 selection quality audit，并比较多个确定性候选策略；不改变运行时工具暴露或执行。

### Modified Capabilities

无。现有工具生命周期、allowlist、sandbox、Eval 和 diagnostics replay requirements 不改变；本 change 只新增一个离线 audit capability，并复用这些 owner 的事实与边界。

## Impact

- 新增 `tool`/`diagnosticsreplay` 侧的 audit、fixture、replay、benchmark、测试和 contract gates；具体实现仍保持 provider-neutral。
- 复用既有 `core/types.Tool.JSONSchema()`、tool registry、MCP/manifest/capability、allowlist、sandbox 与 tool lifecycle 的已准入事实，不新增 registry 或控制面。
- 可能新增文档与 OpenSpec capability spec；不新增 runtime 配置键、诊断写入字段、Provider SDK 依赖或 examples/agent-modes 行为。
- 运行时、ModelRequest、Provider request projection、ReAct、scheduler 和 tool execution path 保持不变。
