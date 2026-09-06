## Why

Baymax 已具备 Skill discovery、adapter manifest/capability、hooks/middleware、sandbox/allowlist、readiness admission 与 RuntimeRecorder 等基础能力，但尚未形成一个统一的扩展资源解析、生命周期治理和故障隔离合同。参考 `earendil-works/pi` 的 ResourceLoader/ExtensionRunner 分层后，当前需要把“扩展从哪里来、是否允许激活、能做什么、失败如何收敛、reload 如何回滚”固化为可验证的嵌入式契约，避免未来 Skill、Plugin、Tool、Model 或 Adapter 接入各自发展出平行语义。

## What Changes

- 定义扩展资源的来源、确定性发现顺序、优先级和同名冲突处理。
- 引入扩展 manifest 投影，描述资源身份、版本兼容范围、内容 digest 和声明能力。
- 固化 requested capability 与 declared capability 的协商，以及激活前的 readiness/policy admission。
- 定义扩展 Hook/工具的超时、资源上限、阻断、失败分类、局部隔离和 finalize 语义。
- 定义 reload/rollback 生命周期：旧实例失效、新实例接管，正在执行的 Run 不被改写，旧实例事件不得回流。
- 将扩展加载、拒绝、降级、阻断、运行失败和恢复信息通过 `RuntimeRecorder` 进入 additive diagnostics，并支持 replay 验证。
- 保持 Run/Stream 语义对等，扩展决策不得创建第二套终态、恢复或 Session/Artifact 事实源。
- **不引入** npm/package manager、托管扩展市场、远程控制面、跨租户调度或平台化 Plugin Registry。

## Capabilities

### New Capabilities

- `extension-lifecycle-governance`: 扩展 manifest、能力协商、激活准入、Hook 生命周期、失败隔离、reload/rollback、诊断与回放合同。
- `deterministic-resource-resolution`: 扩展/Skill/Prompt 等资源的来源分层、优先级、冲突解析、digest 和可重复发现合同。

### Modified Capabilities

无。现有 hooks/middleware、adapter manifest/capability、readiness admission 与 runtime config/diagnostics 合同作为实现依赖复用；本变更只新增扩展治理与资源解析的独立规范，避免在归档时复制既有完整 requirement block。

## Impact

- 代码边界：`skill/loader`、`adapter/manifest`、`adapter/capability`、hooks/middleware、`runtime/config`、readiness、`observability/event` 与相关 composer/runner 接缝。
- 测试与门禁：新增 manifest/资源解析/能力协商/激活拒绝/reload/失败隔离/replay/Run-Stream parity 套件，并接入 shell/PowerShell quality gate。
- 文档与示例：更新模块边界、配置诊断、主线 contract index、路线图和 agent-mode 示例；Example Impact Assessment：`新增示例`。
- 兼容性：只增加 additive、nullable、default 诊断和可选配置；非法配置或非法热更新必须 fail-fast 并原子回滚。
