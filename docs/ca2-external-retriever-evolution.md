# CA2 External Retriever Evolution

更新时间：2026-03-13

## 目标

在保持当前 `http` 兜底可用性的前提下，逐步演进 CA2 Stage2 external retriever，避免过早绑定供应商，同时为性能与语义增强预留路径。

## 分阶段计划

- E1（R3 收敛）：保持 `http` 作为 `rag/db/elasticsearch` 默认兜底实现，优先稳定 SPI 主路径与配置兼容性。
- E1（R3 收敛）：补充 provider profile 模板（request/response mapping、鉴权头、错误字段）降低接入复杂度。
- E2（R4 前半，已完成观测侧）：完善错误分层映射（transport/protocol/semantic，允许新增枚举），并统一映射到 diagnostics 扩展字段与 provider 趋势聚合。
- E2（R4 前半，部分待续）：在 SPI 增加 capability/hint 扩展口，不改 assembler 主流程，为专用 adapter 预留协议位。
- E3（R4 后半，按需触发）：仅在出现性能或语义瓶颈时引入 provider 专用 adapter（例如 ES DSL、向量过滤、rerank 元信息透传）。

## 触发门槛

- `stage2` 路径 P95 延迟持续超阈值。
- `stage2` 路径错误率持续超阈值。
- 业务侧明确要求 provider 专属检索能力（当前通用 HTTP + JSON mapping 无法覆盖）。

## 观测与治理

- 增加 provider 维度看板：命中率、P95、失败原因分布。
- 将触发阈值配置纳入 runtime 观测治理策略，作为是否引入专用 adapter 的决策依据。

## 关联文档

- `docs/development-roadmap.md`
- `docs/context-assembler-phased-plan.md`
