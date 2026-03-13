## Context

CA2 已具备 Stage1/Stage2 路由与 tail recap，但 Stage2 在生产意义上仍是 file-first，`rag/db` 仅占位，导致 Knowledge 路线无法向外部检索系统扩展。用户已确认本期方向：支持 `file/http/rag/db/elasticsearch`，统一通过 JSON 映射与通用接口接入，避免供应商绑定，同时保持现有 fail-fast/best-effort 语义与 library-first 定位。

## Goals / Non-Goals

**Goals:**
- 为 CA2 Stage2 提供统一 Retriever SPI（请求/响应/错误归一）。
- 落地 `http` adapter，并通过 JSON 映射配置接入外部检索服务。
- 让 `rag/db/elasticsearch` provider 在本期具备可运行路径（基于 SPI），不再返回 not-ready。
- 保持 `env > file > default` 配置优先级与 fail-fast/best-effort 语义一致。
- 新增最小 diagnostics 字段：`stage2_hit_count`、`stage2_source`、`stage2_reason`。

**Non-Goals:**
- 不实现任何特定供应商专用 SDK 适配（GraphRAG/RAGFlow/Milvus 等）。
- 不引入新的控制面（Control Plane）或分布式编排能力。
- 不改动 runner 主状态机与既有模型事件对外契约。

## Decisions

### Decision 1: Stage2 统一为 Retriever SPI
- Choice: 定义统一检索接口（query/session/run/top_k/timeout/metadata -> chunks/source/reason/error）。
- Rationale: 屏蔽 provider 差异，避免 assembler 出现多分支耦合。
- Alternative: 每个 provider 直接在 assembler 中实现。Rejected：扩展成本高且不可维护。

### Decision 2: 通用 HTTP adapter + JSON 映射
- Choice: `http` provider 通过可配置 JSON 路径映射请求/响应字段，不固定第三方 schema。
- Rationale: 支持自建 GraphRAG/RAGFlow 类服务，不绑定单一协议细节。
- Alternative: 直接做 GraphRAG/RAGFlow 定制适配。Rejected：会提前锁定供应商。

### Decision 3: rag/db/elasticsearch 先走同一抽象实现
- Choice: 将 `rag/db/elasticsearch` 接到统一 SPI 路径，可通过同构配置指向 HTTP 检索服务或本地 mock 检索源。
- Rationale: 满足“本期可运行”与“未来可替换”两者平衡。
- Alternative: 继续 not-ready。Rejected：不满足本提案目标。

### Decision 4: 复用现有 Stage2 策略
- Choice: 继续使用 CA2 的 `timeout` 与 `stage_policy`，不新增独立重试语义。
- Rationale: 降低行为变化风险，便于回归验证。
- Alternative: 为检索层新增重试策略。Rejected：会扩大语义面并增加调参复杂度。

### Decision 5: 鉴权同时支持 Bearer 与自定义 Header
- Choice: 配置支持 `Authorization: Bearer` 与多自定义头，敏感值由现有 redaction 管线处理。
- Rationale: 覆盖主流接入场景，并与安全基线一致。
- Alternative: 仅支持 Bearer。Rejected：不足以覆盖企业网关/代理场景。

## Risks / Trade-offs

- [Risk] JSON 映射配置复杂，易出现运行时映射错误。 -> Mitigation: 启动/热更新时做强校验并 fail-fast。
- [Risk] 外部检索服务返回不稳定格式。 -> Mitigation: 响应解析做严格结构校验并输出标准化错误原因。
- [Risk] provider 增多可能引入诊断字段语义漂移。 -> Mitigation: 固定字段集合与取值规范，并补契约测试。

## Migration Plan

1. 定义 Retriever SPI 与 provider 归一化结果结构。
2. 扩展配置模型，加入 Stage2 provider 连接与映射参数。
3. 实现 `http/rag/db/elasticsearch` provider 路径并接入 assembler。
4. 扩展 diagnostics 写入与查询字段。
5. 补齐单元与最小集成测试（mock HTTP）。
6. 更新 README/docs 并完成一致性检查。

Rollback strategy:
- 如新 provider 路径不稳定，可临时回退到 `file` provider 作为默认路径。
- 保持配置兼容，确保禁用新 provider 时行为可退回当前基线。

## Open Questions

- `rag/db/elasticsearch` 默认实现是否都先指向通用 HTTP adapter，还是区分最小本地实现路径。
- JSON 映射语法使用 JSONPath 风格还是简化点路径（建议先简化点路径，后续扩展）。
