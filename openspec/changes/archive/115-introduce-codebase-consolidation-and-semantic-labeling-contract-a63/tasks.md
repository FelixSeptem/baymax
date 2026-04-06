## 1. Inventory and Governance Baseline

- [x] 1.1 盘点活动目录中 `ca1|ca2|ca3|ca4|A[0-9]{2,3}` 出现位置（代码、测试、脚本、文档、示例）。
- [x] 1.2 固化 A63 受治理目录矩阵：仅 `openspec/**` 允许 `Axx`；非 `openspec/**` 全量禁止 `Axx`（内容、路径、文件名）。
- [x] 1.3 建立并提交唯一“语义名称 <-> 历史编号/旧名”映射源，作为后续替换与校验基线。

## 2. Context Assembler Semantic Naming Convergence

- [x] 2.1 将活动实现与注释中的 Context Assembler 编号化术语替换为语义化命名。
- [x] 2.2 更新相关单测、集成测试、基准测试命名与描述，移除 `ca1|ca2|ca3|ca4` 活动口径。
- [x] 2.3 为 `cmd/*` 与 `scripts/*` 中历史编号化入口补充语义别名、迁移提示与帮助文本。
- [x] 2.4 对外行为保持不变：验证重命名后 Run/Stream 与既有 contract 语义等价。

## 3. Axx Wording Elimination Outside OpenSpec

- [x] 3.1 在非 `openspec/**` 的代码、测试、脚本、文档、示例中移除散落 `Axx` 文本并替换为语义描述。
- [x] 3.2 清理非 `openspec/**` 中包含 `Axx` 的文件路径与文件名（含目录名、文件名）并完成语义化重命名。
- [x] 3.3 将编号映射收敛到 `openspec/**` 索引层，删除非 `openspec/**` 中的重复编号映射定义。
- [x] 3.4 更新脚本帮助信息与 README 描述，确保用户入口默认展示语义名称。

## 4. Compatibility Bridge and Rollback Safety

- [x] 4.1 为受影响公开配置键提供 alias/迁移跳板并补充优先级与回滚测试。
- [x] 4.2 为受影响诊断字段与解析路径提供兼容读取策略，保持 additive 兼容。
- [x] 4.3 为受影响脚本入口与测试夹具提供兼容别名，避免迁移期调用中断。
- [x] 4.4 编写并验证回滚路径：发生回归时可通过 alias/映射快速恢复旧入口语义。

## 5. Temporary and Outdated Asset Consolidation

- [x] 5.1 清理或归档临时文档、草稿目录与过时占位描述，仅保留当前现状信息。
- [x] 5.2 收敛离线生成物（如离线 scaffold 产物）为“最小可复现样本 + 索引说明”。
- [x] 5.3 将代码中的长期 TODO/future milestone 占位迁移到可追踪 roadmap/index 位置。

## 6. README and Canonical Documentation Paths

- [x] 6.1 同步 `README.md` 与核心模块 README，使其仅描述当前主线路径与现状。
- [x] 6.2 固化架构约束与文档路径入口（roadmap、module boundaries、contract index、config/diagnostics）。
- [x] 6.3 修复 README/模块文档中的非 canonical 路径引用，避免重复或失效入口。
- [x] 6.4 新增/更新 `docs/runtime-harness-architecture.md`，统一描述 `state surfaces -> guides/sensors -> tool mediation -> entropy control` 与 contract/gate 映射。
- [x] 6.5 将 root/module README 对运行时架构入口统一收敛到上述 canonical 文档，并补 docs consistency 校验。

## 7. Naming Regression Gate and Docs Consistency Wiring

- [x] 7.1 新增命名回流扫描脚本（shell/PowerShell）：阻断 `ca1|ca2|ca3|ca4` 回流，并阻断 `A[0-9]{2,3}` 在非 `openspec/**` 的内容/路径/文件名回流。
- [x] 7.2 在扫描脚本中实现单一规则：`openspec/**` 允许 `Axx`，非 `openspec/**` 禁止 `Axx`（内容、路径、文件名）。
- [x] 7.3 将命名扫描与映射一致性校验接入 `check-quality-gate.*` 与 docs consistency 流程。
- [x] 7.4 验证 shell/PowerShell 失败传播语义等价（required native command 非零即 fail-fast）。

## 8. Contract and Test Coverage Alignment

- [x] 8.1 对本提案影响的 capability contract 补齐对应测试覆盖（单测/集成/回放）。
- [x] 8.2 增加“命名收敛不改变语义”的回归测试（Run/Stream parity、replay idempotency、taxonomy stability）。
- [x] 8.3 确保新增/修改 contract 均有可追踪测试用例与 gate 映射。

## 9. Validation, Risk Record, and Delivery

- [x] 9.1 执行最低验证：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`。
- [x] 9.2 执行门禁：`check-quality-gate.*`、`check-docs-consistency.*` 与 A63 命名治理脚本。
- [x] 9.3 记录未执行项、风险点、回滚点与迁移影响，确保评审可追溯。
- [x] 9.4 更新 roadmap 与主线索引中的 A63 状态与验收口径，保持状态口径一致。

## 10. Runtime Config and Diagnostics Contract Migration

- [x] 10.1 盘点 `runtime/config`、`runtime/diagnostics`、`core/runner` payload 中 CA-era 键/字段并定义语义主名映射。
- [x] 10.2 实现“语义主名 + 兼容旧名”迁移窗口（双读或别名），明确冲突优先级与迁移提示输出。
- [x] 10.3 补充 parser compatibility 测试（legacy-only、semantic-only、mixed 输入）并验证 additive 兼容。
- [x] 10.4 同步更新 `docs/runtime-config-diagnostics.md` 字段映射与迁移窗口说明。

## 11. Replay and Identifier Semanticization

- [x] 11.1 为命名迁移补齐 replay mixed-fixture 套件，确保历史夹具与语义新名共存可回放。
- [x] 11.2 将 non-`openspec/**` 下 gate/test fixture/env key 中 `Axx`/`CAx` 标识替换为语义标识；`Axx` 历史映射仅保留在 `openspec/**`。
- [x] 11.3 更新 `scripts/check-*.sh/.ps1` 中硬编码编号断言与测试过滤表达式，避免 gate 级编号耦合残留。
- [x] 11.4 验证 replay drift taxonomy 在命名迁移前后稳定，不引入分类漂移。

## 12. Additional Cleanup Removal

- [x] 12.1 清退 `examples/adapters/_a23-offline-work/*` 冗余副本，仅保留最小可复现样本与索引。
- [x] 12.2 归档或替换过时阶段文档（含 `docs/context-assembler-phased-plan.md`、`docs/v1-acceptance.md`）为“当前现状 + 索引”描述。
- [x] 12.3 清理并阻断临时备份工件（例如 `*.go.<timestamp>`、临时草稿副本）进入活动目录。
- [x] 12.4 将离线产物保留策略与仓库卫生规则接入 docs consistency 与 quality gate。

## 13. Large File Split Governance

- [x] 13.1 盘点 non-`openspec/**` 下 `*.go` 文件行数，形成超限清单并标注优先级（核心运行路径优先）。
- [x] 13.2 定义并落地单文件行数预算（默认阈值、硬阈值、统计口径）及受控例外清单格式（owner/reason/expiry）。
- [x] 13.3 对超限核心文件执行语义拆分（提取子文件/子模块），保持导出行为与 contract 语义不变。
- [x] 13.4 新增 `*.go` 行数预算检查脚本（shell/PowerShell）并接入 `check-quality-gate.*`。
- [x] 13.5 增加“超限债务不扩张”校验：已有超限文件若继续增长且无有效例外则阻断。
- [x] 13.6 将 `*.go` 拆分变更纳入“语义不变强校验”路径：Run/Stream parity、replay idempotency、impacted contract suites 必跑。
- [x] 13.7 为 `*.go` 拆分场景新增 gate 阻断规则：任一强校验失败直接 fail-fast，不允许软通过。
- [x] 13.8 补充拆分回归测试用例，证明拆分前后逻辑分支、终态语义和 drift classification 保持一致。
