# External Adapter Template Index (A21)

更新时间：2026-03-19

## 目标

提供外部接入的最小模板入口，降低新接入方在 `MCP / Model / Tool` 三类适配上的理解与迁移成本。

优先级（固定）：
1. MCP adapter template
2. Model provider adapter template
3. Tool adapter template

模板边界：
- 仅用于 onboarding skeleton 和迁移参考。
- 不提供生产级运行保障（多租户隔离、SLO、审计、全量安全策略）。

## 模板导航

| 优先级 | 模板 | 位置 | 运行命令 |
| --- | --- | --- | --- |
| P1 | MCP adapter template | `examples/templates/mcp-adapter-template/main.go` | `go run ./examples/templates/mcp-adapter-template` |
| P2 | Model provider adapter template | `examples/templates/model-adapter-template/main.go` | `go run ./examples/templates/model-adapter-template` |
| P3 | Tool adapter template | `examples/templates/tool-adapter-template/main.go` | `go run ./examples/templates/tool-adapter-template` |

## MCP Adapter Template（P1）

最小可运行片段（`examples/templates/mcp-adapter-template/main.go`）展示：
- `mcp/stdio` 客户端接入；
- transport 协议骨架（`Initialize/ListTools/CallTool/Close`）；
- 本地最小调用链路。

边界说明：
- 模板使用 `fakeTransport` 演示结构，不代表真实网络传输实现。
- 生产接入时需要补齐：鉴权、重试预算、连接生命周期、告警与审计。
- 适配层不应穿透 `core/runner` 内部状态机语义。

## Model Adapter Template（P2）

最小可运行片段（`examples/templates/model-adapter-template/main.go`）展示：
- 实现 `types.ModelClient`（`Generate/Stream`）；
- 通过 `runner.New` 进入统一 Run 链路。

边界说明：
- 模板聚焦接口收敛，不覆盖 provider SDK 的生产化治理策略。
- provider-specific 协议映射应保持在 `model/<provider>` 子域，不泄漏到 `core/*`。
- 生产接入需补齐错误分类与 capability discovery。

## Tool Adapter Template（P3）

最小可运行片段（`examples/templates/tool-adapter-template/main.go`）展示：
- 实现 `types.Tool`；
- 通过 `tool/local.Registry` 注册；
- 在 runner 闭环中触发工具调用并收敛终态。

边界说明：
- 模板为单工具骨架，不包含复杂 schema 演进与安全过滤策略。
- 工具超时、限流、幂等与副作用控制需由业务侧补齐。
- 生产场景建议结合 `runtime/security` 与 action gate 规则治理。

## 相关文档

- 迁移映射：`docs/adapter-migration-mapping.md`
- API 参考入口：`docs/api-reference-d1.md`
- 运行时配置与诊断：`docs/runtime-config-diagnostics.md`

## Conformance 验收入口（A22）

模板交付后，必须通过 A22 一致性验收：

```bash
bash scripts/check-adapter-conformance.sh
```

```powershell
pwsh -File scripts/check-adapter-conformance.ps1
```

验收覆盖：
- MCP/Model/Tool 最小矩阵（优先级 `MCP > Model > Tool`）
- 默认离线执行（stub/fake）
- run/stream 语义等价（适用项）
- 错误分类与 reason taxonomy 归一
- mandatory input fail-fast
