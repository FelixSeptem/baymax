## Why

当前主线契约已经把 mailbox 作为 sync/async/delayed 协调的 canonical 路径，但 `orchestration/invoke` 仍暴露并内部复用 direct `InvokeSync`/`InvokeAsync` 入口，形成“已弃用但仍被主路径依赖”的中间态。  
在 A33 收口协作重试后，应继续消除该中间态，把调用面收敛为单一 mailbox canonical 入口，降低语义漂移和回归风险。

## What Changes

- 将多代理调用主线入口固定为 `orchestration/mailbox` + `orchestration/invoke/mailbox_bridge`。
- **BREAKING**：退场 legacy direct invoke 对外入口（`invoke.InvokeSync`、`invoke.InvokeAsync`），不再作为公共 canonical API。
- 重构 `MailboxBridge` 内部执行路径，去除对 deprecated 导出函数的依赖，改为私有实现收敛。
- 在 shared multi-agent gate 与 quality gate 增加 canonical-only 阻断，防止 legacy direct 入口回流。
- 同步更新 README/roadmap/主干契约索引与 orchestration 模块文档，删除中间态叙述，仅保留最新状态。

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `multi-agent-sync-invocation-contract`: 从“mailbox canonical + direct deprecated”收敛到“mailbox 唯一同步调用入口”语义。
- `multi-agent-async-reporting`: 从“legacy direct report-sink deprecated”收敛到“legacy direct async invoke 退出公共契约面”语义。
- `multi-agent-mailbox-contract`: 强化 mailbox 作为 sync/async/delayed 统一调用入口的约束。
- `go-quality-gate`: 增加 canonical-only 防回流检查，阻断 legacy direct invoke API 再次暴露/复用。

## Impact

- 代码：
  - `orchestration/invoke/*`
  - `orchestration/collab/*`
  - `orchestration/scheduler/*`
  - `integration/*`（shared multi-agent contract suites）
  - `scripts/check-multi-agent-shared-contract.*`、`scripts/check-quality-gate.*`
- 文档：
  - `README.md`
  - `docs/development-roadmap.md`
  - `docs/mainline-contract-test-index.md`
  - `orchestration/README.md`
- 兼容性：
  - `0.x` 阶段允许能力与接口收敛，A34 明确采用 break-and-cleanup 路线，不保留 legacy direct invoke 对外兼容层。
