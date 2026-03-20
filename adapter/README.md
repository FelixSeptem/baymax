# adapter 组件说明

## 功能域

`adapter` 是 Baymax 外部接入契约域，负责“接入前声明 + 运行时协商 + 脚手架生成”的统一收敛：

- `adapter/manifest`：manifest 解析、校验、运行时兼容激活
- `adapter/capability`：能力协商与降级策略
- `adapter/scaffold`：外部适配脚手架与契约测试骨架生成

该域的目标是让外部 adapter 在 `MCP / Model / Tool` 三类接入上保持一致的 fail-fast 与 downgrade 语义，而不是把契约散落到业务代码。

## 架构设计

`adapter` 内部按“静态合同 -> 运行时协商 -> 交付骨架”三层组织：

1. 静态合同层（`manifest`）
- 负责 `adapter-manifest.json` 的 schema 约束与 semver 兼容判断。
- 接入边界提供 deterministic 错误分类（missing/invalid/mismatch）。

2. 协商层（`capability`）
- 统一 requested-vs-declared 匹配。
- 固定策略集合 `fail_fast | best_effort`，并输出标准 reason taxonomy。

3. 交付层（`scaffold`）
- 生成最小可运行的 adapter 目录结构。
- 默认携带 manifest、conformance bootstrap、negotiation baseline 测试骨架。

## 关键入口

- `adapter/manifest/manifest.go`
- `adapter/capability/negotiation.go`
- `adapter/scaffold/scaffold.go`
- `integration/adapterconformance/harness.go`（跨模块契约验收入口）

## 边界与依赖

- `adapter/*` 仅承载 adapter 契约与脚手架逻辑，不承载 provider SDK 或 transport 运行时细节。
- `adapter/manifest` 依赖 `adapter/capability` 做协商结果收敛；不直接依赖 `runtime/diagnostics`。
- 诊断与回放仍通过主线事件/contract gate 校验，不在 `adapter/*` 内部自行写入诊断存储。
- 外部接入验收必须通过 `integration/adapterconformance` 与 `scripts/check-*` 门禁，不以“本地示例可跑”替代。

## 配置与默认值

- manifest 必填字段：`type/name/version/baymax_compat/capabilities.required/capabilities.optional/conformance_profile`。
- 协商默认策略：`fail_fast`（当 manifest 未显式配置 `negotiation.default_strategy` 时）。
- 请求级策略覆盖：由 `manifest.negotiation.allow_request_override` 决定是否允许。
- profile version/replay gate 的字段与兼容窗口以当前代码与 OpenSpec 为准。

## 可观测性与验证

- 包级验证：
  - `go test ./adapter/manifest ./adapter/capability ./adapter/scaffold -count=1`
- 交叉契约验证：
  - `go test ./integration/adapterconformance -count=1`
- 门禁脚本：
  - `bash scripts/check-adapter-manifest-contract.sh`
  - `bash scripts/check-adapter-capability-contract.sh`
  - `bash scripts/check-adapter-scaffold-drift.sh`
  - `pwsh -File scripts/check-adapter-manifest-contract.ps1`
  - `pwsh -File scripts/check-adapter-capability-contract.ps1`
  - `pwsh -File scripts/check-adapter-scaffold-drift.ps1`

## 扩展点与常见误用

- 扩展点：新增 capability 族时，先补 reason taxonomy 与 conformance 矩阵，再补运行时接入逻辑。
- 扩展点：新增脚手架模板字段时，同步更新 manifest/profile 对齐校验与 drift gate。
- 常见误用：把 `required` 能力当作可降级能力处理，破坏 fail-fast 边界。
- 常见误用：只改模板不改 conformance harness，导致生成产物与验收口径漂移。
- 常见误用：在 adapter 层直接写业务诊断存储，绕过统一单写路径。
