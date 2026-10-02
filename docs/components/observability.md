# Observability

## Purpose

observability 提供 event recorder、timeline、diagnostics、export 和 replay 所需的可验证事实。

## Owned facts/state

事件 schema、写入顺序、脱敏结果、查询投影和导出 bundle 由 observability owner 管理。

## Inputs/outputs

输入是各组件发出的 runtime events；输出是 bounded timeline、diagnostics query 和 replay fixture。

## Dependencies and forbidden dependencies

允许依赖标准库和稳定的 runtime 抽象；禁止反向调用业务组件改变执行。

## Lifecycle / failure / rollback

事件写入失败必须可观测；导出失败不改变已完成 runtime outcome。fixture 生成不访问网络。

## Observability

本组件本身是诊断单写入口，禁止新增旁路 recorder。

## Verification / source links

`go test ./observability/... ./runtime/diagnostics/... -count=1`；[diagnostics replay](../diagnostics-replay.md)。
