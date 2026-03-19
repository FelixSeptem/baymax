## Context

当前外部接入路径已经具备样板与迁移文档（A21 进行中），但还没有“强制执行的一致性验收层”。这意味着 adapter 作者可以按样板开发，却无法通过统一 gate 证明其语义满足 Baymax 合同边界。A22 需要在不增加平台依赖的前提下，补齐离线可执行的 conformance harness，并将其纳入标准质量门禁。

## Goals / Non-Goals

**Goals:**
- 提供 adapter conformance harness，覆盖 MCP/Model/Tool 三类最小矩阵。
- 固化关键语义检查：Run/Stream 等价、错误归一、降级行为、fail-fast 边界。
- 默认离线执行（stub/fake），确保本地与 CI 稳定复现。
- conformance gate 作为阻断项接入 quality-gate 主路径。

**Non-Goals:**
- 不引入性能阈值或 benchmark 验收（性能由 A19 路径治理）。
- 不新增外部协议或运行时行为变更。
- 不构建平台化 adapter registry/marketplace。

## Decisions

### 1) 验收框架聚焦“语义一致性”，不含性能维度
- 方案：A22 仅验证语义契约，不比较延迟吞吐。
- 原因：与 A19 职责分离，降低 gate 波动。
- 备选：合并性能检查。拒绝原因：职责重叠且噪声更高。

### 2) 默认离线模式，禁止对外部服务硬依赖
- 方案：harness 使用 stub/fake/provider mock 数据集。
- 原因：提升可重复性，避免 CI 环境不确定性。
- 备选：真实 provider 调用。拒绝原因：网络抖动与凭证成本高。

### 3) 失败策略采用 fail-fast 且默认阻断
- 方案：任一 conformance 检查失败即退出非零并阻断质量门禁。
- 原因：外部适配属于兼容边界，必须强约束。
- 备选：warn-only。拒绝原因：无法防止契约漂移。

### 4) 最小矩阵按 `MCP > Model > Tool` 优先级落地
- 方案：先保证 MCP，一次性覆盖 model/tool 的最小可判定场景。
- 原因：对外互联风险最高的路径优先收口。
- 备选：三类同深度全覆盖。拒绝原因：首期成本过高。

### 5) A21 样板必须可被 A22 harness 验收
- 方案：样板文档中的最小代码路径需映射到 conformance case。
- 原因：保证“文档可执行”。
- 备选：样板与 harness 分离演进。拒绝原因：易发生双轨漂移。

## Risks / Trade-offs

- [Risk] conformance 夹具设计过重，增加维护成本  
  → Mitigation: 首期仅覆盖最小矩阵与稳定断言，后续按 A22+ 迭代扩展。

- [Risk] 不同 adapter 对同一错误语义映射不一致  
  → Mitigation: 统一错误层级和 reason code 断言，失败时输出机器可读分类。

- [Risk] A20/A21 并行实施引入脚本冲突  
  → Mitigation: 仅在 quality-gate 接入点做最小增量，保持脚本可组合结构。

## Migration Plan

1. 新增 conformance harness 目录与测试夹具。
2. 实现 MCP/Model/Tool 最小矩阵测试用例。
3. 增加 `check-adapter-conformance.sh/.ps1`。
4. 接入 `check-quality-gate.sh/.ps1` 作为阻断步骤。
5. 更新 mainline index 与 A21 文档入口的 conformance 链接。

回滚策略：
- 若 gate 稳定性不足，可先回滚 quality-gate 接入点；
- harness 与测试可保留为非阻断执行，不影响 runtime 主链路。

## Open Questions

- 当前无阻塞问题，按推荐值冻结：阻断、离线、fail-fast、最小矩阵、A21 样板接入、无性能阈值。
