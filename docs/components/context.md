# Context

## Purpose

`context` 负责 canonical context、预算投影、压缩和 handoff，不负责 Provider SDK 调用。

## Owned facts/state

上下文条目、预算、压缩结果、引用/检索证据和投影元数据由 context owner 管理。

## Inputs/outputs

输入为宿主 context、历史、retriever 和预算；输出为只读 snapshot 与 projection evidence。

## Dependencies and forbidden dependencies

可依赖标准库和 context 内部模块；禁止直接导入 Provider 官方 SDK 或 `mcp/internal`。

## Lifecycle / failure / rollback

先校验预算与 scope，再按策略投影；超预算使用 bounded compaction，失败保留原 snapshot。

## Observability

输出 projection evidence，由 recorder 记录原因、预算和 fallback，不写入 Provider 私有字段。

## Verification / source links

`go test ./context/... -count=1`；[context README](../../context/README.md)、[context design](../context-assembler-phased-plan.md)。
