## Why

`examples/agent-modes` 当前虽然可运行，但大量模式实现仍呈现同构模板形态，难以作为真实业务语义示例使用。需要新增独立提案，以“文档先行 + 分批替换 + 反模板门禁”方式收敛，避免再次出现“任务已勾选但代码仍模板化”的偏差。

## What Changes

- 新增独立提案 `a72`，专门治理 `examples/agent-modes` 的反模板落地，不复用已归档 a62/a71 的任务勾选状态。
- 明确实施顺序：先完成模式级文档基线（语义锚点、真实运行路径、验证证据、失败回滚），再按文档逐模式实现代码。
- 新增反模板强约束：每个模式必须拥有模式自有业务语义执行逻辑，禁止仅通过常量/marker 参数化共用模板骨架。
- 强化 `minimal`/`production-ish` 变体要求：差异必须来自真实行为分支，不允许仅追加占位字段。
- 新增并接入质量门禁：
  - `check-agent-mode-anti-template-contract.sh/.ps1`
  - `check-agent-mode-doc-first-delivery-contract.sh/.ps1`
- 收敛任务验收口径：未完成“代码证据 + 测试证据 + 文档证据 + 门禁证据”四要素的任务禁止勾选。

## Example Impact Assessment

- 修改示例

## Capabilities

### New Capabilities

- `agent-mode-anti-template-doc-first-delivery-contract`: 定义示例文档先行、按模式语义落地与反模板阻断的统一契约。

### Modified Capabilities

- `real-runtime-agent-mode-examples-contract`: 增加“每模式语义自有实现、禁止模板骨架回流、文档先行顺序”的规范要求。
- `go-quality-gate`: 增加反模板与文档先行一致性门禁，要求 shell/PowerShell parity 且失败即阻断。

## Impact

- 示例代码：`examples/agent-modes/*/semantic_example.go`、`examples/agent-modes/*/{minimal,production-ish}/main.go`
- 示例文档：`examples/agent-modes/*/{minimal,production-ish}/README.md`、`examples/agent-modes/MATRIX.md`、`examples/agent-modes/PLAYBOOK.md`
- 质量门禁：`scripts/check-agent-mode-anti-template-contract.*`、`scripts/check-agent-mode-doc-first-delivery-contract.*`、`scripts/check-quality-gate.*`
- 测试与验证：`integration/*`（模式语义回归）、`scripts/check-agent-mode-examples-smoke.*`
