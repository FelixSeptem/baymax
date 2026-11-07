# Baymax 技术文档

本文档按读者任务组织。代码、测试和 `openspec/specs` 是行为事实源；这里负责解释设计、边界、owner 和下一步阅读位置。发生冲突时，以 [事实源与迁移矩阵](documentation-ownership.md) 指向的 canonical source 为准。

## 外部开发者主路径

如果你第一次接入 Baymax，按以下顺序阅读：

1. [根 README](../README.md)：定位、能力边界和最小运行命令。
2. [开始使用](guides/getting-started.md)：前置条件、离线 quickstart、真实 Provider 和宿主接入清单。
3. [架构总览](architecture/overview.md)：宿主、runner、context、model、tool、orchestration 和 observability 如何组合。
4. [Execution Model](architecture/execution-model.md)：Run、Task、Attempt、Event、Checkpoint、Terminal Outcome 和 Business Effect 如何关联。
5. [设计原则](architecture/design-principles.md)：为什么状态由 source owner 持有，为什么不支持的能力必须 fail closed。
6. [运行生命周期](architecture/runtime-lifecycle.md)：一次 Run/Stream 从 admission 到 terminal 的状态路径。
7. [数据与事件流](architecture/data-event-flow.md)：source facts、事件投影、diagnostics、replay 和宿主业务 effect 的关系。
8. [模块 ownership](architecture/module-ownership.md)：每个包负责什么、禁止依赖什么。
9. [集成指南](guides/integration.md)：根据 direct/tool/MCP/provider/orchestration/host bridge 选择接入路径。
10. 对应的 [示例](../examples/01-chat-minimal) 和 [测试/replay/gate 指南](guides/testing-replay-gates.md)：用离线证据验证接入。

## 按任务查找

### 接入与运行

- [集成指南](guides/integration.md)：能力矩阵、owner、默认边界和示例入口。
- [开始使用](guides/getting-started.md)：从离线 fake model 到真实 Provider 的最短接入路径。
- [最佳实践](guides/best-practices.md)：Run/Stream 对等、超时、错误、授权和业务 commit 分界。
- [配置与诊断](guides/configuration-and-diagnostics.md)：`env > file > default`、快照、脱敏和排查顺序。
- [Host resume admission](guides/host-resume-admission.md)：direct native checkpoint 的宿主重验证、duplicate、CAS 和 fail-closed 边界。
- [外部 adapter 模板索引](external-adapter-template-index.md)：Provider、tool、MCP 和 sandbox adapter 的 manifest、capability negotiation 与 conformance 路径。
- [Adapter 迁移映射](adapter-migration-mapping.md)：从旧接入模式迁移到显式 capability/manifest 的兼容与回滚规则。
- [API 参考覆盖](api-reference-d1.md)：外部集成主路径的导出 API、最小用途和示例入口。

### 理解组件

- [组件责任索引](components/README.md)：runner、context、provider、tool、MCP、orchestration、runtime、observability、adapter、extension 和 skills。
- [运行时模块边界](runtime-module-boundaries.md)：允许/禁止的依赖方向。
- [Execution Model](architecture/execution-model.md)：跨组件的运行时概念、身份和恢复边界。
- [Runtime Harness 架构](runtime-harness-architecture.md)：状态表面、工具中介、治理和 gate 的总览。
- [执行图拓扑投影](guides/execution-graph.md)：workflow、team 与 child dispatch 的 source-neutral topology projection。

### 验证与贡献

- [测试、回放与门禁](guides/testing-replay-gates.md)：unit、integration、contract、replay、gate 的分工。
- [Diagnostics Replay](diagnostics-replay.md)：可脱敏、bounded、离线的证据回放。
- [贡献指南](guides/contribution.md)：分支、验证、PR、回滚和未执行项记录。
- [OpenSpec 工作流](guides/openspec-workflow.md)：提案、文档影响评估和归档。
- [故障排查](guides/troubleshooting.md)：状态漂移、文档门禁、配置和文件占用问题。

## Reference and history map

这些页面不是另一套运行时事实源；它们按用途分组，帮助外部开发者判断“应当依赖什么”以及“哪些内容只用于历史背景”。

### Canonical reference

- [Runtime 配置与诊断事实源](runtime-config-diagnostics.md)：配置键、默认值、校验、回滚和公开诊断字段。
- [Runtime 模块边界](runtime-module-boundaries.md)：允许/禁止的包依赖方向。
- [主线契约测试索引](mainline-contract-test-index.md)：公开能力到 unit/integration/replay/gate 的证据映射。
- [Diagnostics Replay](diagnostics-replay.md)：有界、脱敏、只读的离线证据回放。
- [MCP runtime profiles](mcp-runtime-profiles.md)：MCP transport/profile 的配置与诊断入口。
- [多 agent 标识模型](multi-agent-identifier-model.md)：parent/child/member/task/attempt 的标识和关联规则。
- [Action capability audit owner map](action-capability-audit-owner-map.md)：action capability 审计的 source ownership 边界。
- [Evaluation contract](evaluation-contract.md)：离线候选比较的身份、隔离、成本和 replay 边界。
- [Versioning and compatibility](versioning-and-compatibility.md)：pre-1.x 版本阶段、兼容性和迁移口径。
- [API 参考覆盖](api-reference-d1.md)：`core/types`、`core/runner`、`runtime/config`、`runtime/diagnostics`、`context/assembler` 和 `skill/loader` 的公开入口。

### Source-owner package references

这些 README 是实现 owner 的事实源，补充组件概览但不复制完整 contract：

- [Memory owner](../memory/README.md)：memory SPI、filesystem WAL/snapshot、scope、fallback 和 lifecycle。
- [Native event adapter](../host/eventadapter/README.md)：实时 callback projection、source journal 边界和 sink failure 语义。
- [Context handoff](../context/handoff/README.md) 与 [host context projection](../context/hostprojection/README.md)：reference-only assembly/handoff 与 current-authority resolver 边界。
- [Integration verification](../integration/README.md)：跨模块 contract、replay、adapter 和 sandbox conformance 套件。

### Historical or planning material

- [V1 acceptance baseline](v1-acceptance.md)：早期能力清单，仅作已知限制快照，不覆盖当前 capability manifest。
- [Modular E2E review matrix](modular-e2e-review-matrix.md)：评审矩阵与缺口记录，不是运行时 API。
- [Context assembler phased plan](context-assembler-phased-plan.md)、[stage-2 retriever evolution](context-stage2-external-retriever-evolution.md) 和 [examples expansion plan](examples-expansion-plan.md)：规划材料，实施状态以 roadmap/OpenSpec/archive 为准。
- [Pigo agent design study](pigo-agent-design-study.md) 与 [Pi comparison/adoption study](pi-agent-comparison-and-adoption-study.md)：外部项目调研和借鉴边界，不代表 Baymax 已实现对应模型。

如果一个历史/规划页面与代码、测试或 [documentation ownership map](documentation-ownership.md) 冲突，应以当前 source owner 和契约证据为准；不要把研究文档中的 future direction 当成已支持能力。

## 架构与事实源

- [事实源与迁移矩阵](documentation-ownership.md)：canonical source、入口、owner 和验证命令。
- [Owner 证据索引](owner-evidence-index.md)：已登记语义的唯一 owner 与验证链。
- [文档写作与页面模板](documentation-style-guide.md)：术语、页面结构、链接和弃用规则。
- [主线契约测试索引](mainline-contract-test-index.md)：公开契约到测试/gate 的映射。
- [运行时配置与诊断](runtime-config-diagnostics.md)
- [外部适配模板索引](external-adapter-template-index.md)
- [适配迁移映射](adapter-migration-mapping.md)
- [版本与兼容](versioning-and-compatibility.md)
- [开发路线图](development-roadmap.md)

## 设计阅读提示

Baymax 是嵌入式 runtime，不是托管控制面。宿主拥有凭据、网络、持久化、业务授权和最终 effect；Baymax 负责可组合的执行 primitive、标准事件、source-owned lifecycle 和可验证证据。读取某个能力时，先确认 owner 和 capability manifest，再确认是否 durable、是否可 replay、是否能 restore；不要从 DTO 存在、package 名称或 callback 事件推断能力。
