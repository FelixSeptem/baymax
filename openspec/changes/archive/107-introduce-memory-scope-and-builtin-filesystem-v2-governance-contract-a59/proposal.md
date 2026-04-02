## Why

A58 正在收敛跨策略判定，但 memory 侧仍存在三类未冻结契约：scope 解析不统一、写入/检索/生命周期策略缺少统一治理、builtin filesystem 在索引恢复与质量回归上缺少 v2 合同。当前缺口会直接导致 memory 注入不可解释、跨 provider 接入成本上升、以及回放门禁无法稳定拦截检索退化。

为避免在 memory 方向继续拆分平行提案，A59 需要一次性将 `scope + mode + search + lifecycle + filesystem v2` 合并为主线 contract。

## What Changes

- 新增 A59 主合同：memory scope + builtin filesystem v2 governance。
- 冻结 memory scope 解析语义：`session|project|global`，并补齐注入预算裁剪规则。
- 冻结 memory 写入模式：`automatic|agentic`，明确回填窗口与幂等约束（新增 `runtime.memory.write_mode.*`，不复用既有 backend 选择字段 `runtime.memory.mode`）。
- 冻结检索治理链路：`hybrid retrieval + rerank + temporal decay` 的最小可配置合同。
- 新增 memory lifecycle 合同：`retention|ttl|forget` 的 fail-fast 校验与执行可观测。
- 补齐 builtin filesystem v2：索引增量更新、全量重建触发、WAL/snapshot 恢复一致性与 drift detect。
- 新增 QueryRuns additive 字段：`memory_scope_selected`、`memory_budget_used`、`memory_hits`、`memory_rerank_stats`、`memory_lifecycle_action`。
- 新增 replay fixtures：`memory_scope.v1`、`memory_search.v1`、`memory_lifecycle.v1`，并冻结 drift taxonomy。
- 新增 contract gate：`check-memory-scope-and-search-contract.sh/.ps1`，并接入质量门禁。
- 同步 roadmap、runtime config/diagnostics 文档、memory README 与主线 contract index。

## Capabilities

### New Capabilities
- `memory-scope-and-builtin-filesystem-v2-governance-contract`: 冻结 memory scope、写入模式、检索治理、生命周期与 builtin filesystem v2 的统一契约。

### Modified Capabilities
- `runtime-memory-engine-spi-and-filesystem-builtin`: 扩展 builtin filesystem v2 的索引更新、恢复一致性与 fallback 可观测要求。
- `runtime-config-and-diagnostics-api`: 增加 `runtime.memory.scope|write_mode|injection_budget|lifecycle|search` 配置域与 memory additive 诊断字段。
- `diagnostics-replay-tooling`: 增加 memory scope/search/lifecycle fixtures 与 drift 分类断言。
- `go-quality-gate`: 增加 memory contract gate 及 required-check 暴露。

## Impact

- 代码：
  - `memory/*`（scope/mode/search/lifecycle 合同与 builtin filesystem v2）
  - `runtime/config`、`runtime/diagnostics`、`observability/event`（配置与 additive 字段）
  - `tool/diagnosticsreplay`、`integration/*`（memory fixtures + drift tests）
  - `scripts/check-memory-scope-and-search-contract.*`、`scripts/check-quality-gate.*`
- 文档：
  - `docs/development-roadmap.md`
  - `docs/runtime-config-diagnostics.md`
  - `docs/mainline-contract-test-index.md`
  - `memory/README.md`、`README.md`
- 兼容性：
  - 对外 API 保持兼容；新增字段遵循 `additive + nullable + default`。
  - 不引入平台化 memory 控制面或托管服务，保持 `library-first` 形态。
