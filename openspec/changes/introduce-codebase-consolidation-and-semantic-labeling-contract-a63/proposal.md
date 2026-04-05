## Why

A67-CTX 与 A68 进入实施后，主干能力边界已经较完整，但仓库仍存在历史编号命名与临时文档资产并行的问题：`ca1/ca2/ca3/ca4` 命名口径在活动目录持续外溢，`Axx` 编号字眼散落于代码、测试、脚本与文档，导致认知负担高、审查成本高、回归口径不集中。此时启动 A63，可在不改变运行时语义前提下完成命名与文档治理收口，为后续 A64 性能治理和 A62 交付易用性提供稳定基线。

## What Changes

- 新增 A63 主合同：代码库收敛与语义化命名治理（codebase consolidation + semantic labeling）。
- 强制范围 1：Context Assembler 统一命名。
  - 在活动代码、测试、脚本、文档中统一为语义化名称；
  - 收敛后不再将 `ca1`、`ca2`、`ca3`、`ca4` 作为模块/阶段命名口径。
- 强制范围 2：消除 `Axx` 字眼并替换为语义化描述。
  - 覆盖代码、测试、脚本、文档、示例说明、文件路径、文件名；
  - `Axx` 仅允许存在于 `openspec/**`（内容、文件路径、文件名）；`openspec/**` 之外一律禁止。
- 临时/过时资产治理：清理或归档临时文档、离线生成物与占位内容，仅保留可复现最小样本与索引说明。
- 阶段工具命名治理：为 `cmd/*` 与 `scripts/*` 中编号化入口补齐语义别名与帮助信息，并建立统一迁移映射。
- 兼容桥接与回滚：对公开配置键、诊断字段、脚本入口、测试夹具提供 alias/迁移跳板与回滚策略，保证语义不变、行为不变。
- 契约面命名迁移治理：对 `runtime/config`、`runtime/diagnostics`、`core/runner` payload、replay fixture 中历史编号字段建立“语义主名 + 兼容旧名”迁移窗口，明确默认输出、兼容读取与下线策略。
- 脚本与测试标识语义化：将 gate/test fixture/env key 中历史 `Axx`/`CAx` 标识替换为语义化标识；涉及 `Axx` 的历史追溯统一转入 `openspec/**`。
- 回流阻断：新增/加强命名治理扫描门禁（shell/PowerShell 等价），阻断 `ca1|ca2|ca3|ca4` 在活动目录回流，并阻断 `A[0-9]{2,3}` 在 `openspec/**` 之外的任意内容/路径/文件名回流。
- 语义词表集中化：维护唯一“语义名称 <-> 历史编号/旧名”映射表（位于 `openspec/**`），供代码注释、README、脚本帮助与测试命名统一引用。
- 冗余资产清退：删除或归档离线 scaffold 批量副本、过时阶段性文档与临时备份工件（如 `*.go.<timestamp>`），并纳入仓库卫生门禁。
- 过大文件拆分治理：为非 `openspec/**` 的 `*.go` 文件引入单文件行数预算与超限拆分策略，避免“巨型文件”持续增长并降低评审/回归复杂度（其他文件类型暂不纳入）。
  - 拆分后必须保证语义不变、逻辑不变；通过强校验门禁（contract/replay/Run-Stream parity）后方可合入。

## Capabilities

### New Capabilities
- `codebase-consolidation-and-semantic-labeling-contract`: 定义命名统一、编号语义化替换、临时资产清理、兼容桥接、回流阻断与集中映射的收敛合同。

### Modified Capabilities
- `context-assembler-production-convergence`: 收敛 Context Assembler 命名口径，禁止 `ca1|ca2|ca3|ca4` 作为活动命名并补齐语义映射引用。
- `core-module-readme-richness`: 规范 README/模块文档仅描述当前现状，移除临时与过时条目并对齐集中索引。
- `go-quality-gate`: 增加命名回流阻断与编号散落扫描校验，要求 shell/PowerShell 门禁语义等价。
- `go-quality-gate`: 增加大文件行数预算检查与超限阻断（含受控例外清单）。
- `go-quality-gate`: 对 `*.go` 拆分重构启用语义不变强校验，任一强校验失败即阻断。
- `runtime-config-and-diagnostics-api`: 为历史 `ca*` 配置键与诊断字段提供语义化主名、兼容别名读取与 parser 兼容测试要求。
- `diagnostics-replay-tooling`: 对命名迁移造成的字段变化补充 mixed-fixture 兼容断言，确保 replay 语义与 drift 分类不漂移。

## Impact

- 代码与脚本：
  - `context/*`、`core/*`、`runtime/*`、`tool/*`、`model/*`、`orchestration/*` 中涉及历史命名/编号描述的注释、测试名、帮助文本与入口映射。
  - `scripts/*`、`cmd/*` 的编号化入口别名、迁移提示与门禁接线。
  - 超限 `*.go` 大文件的语义拆分重构（提取子模块/辅助文件），并保持外部行为与契约语义不变。
  - `integration/*`、`examples/*`（尤其 `examples/adapters/_a23-offline-work/*`）的离线产物收敛与编号标识清退。
- OpenSpec 与文档：
  - `openspec/**` 作为唯一允许保留 `Axx` 字样的目录（内容、路径、文件名）。
  - `README.md`、`AGENTS.md`、`docs/*`（尤其 roadmap、模块边界、配置诊断、主线合同索引）同步语义化与现状化收敛。
  - 清理或归档过时阶段文档（如 `docs/context-assembler-phased-plan.md`、`docs/v1-acceptance.md`）。
- 兼容与风险控制：
  - 不改变 Run/Stream、readiness/admission、reason taxonomy、diagnostics/replay 合同语义。
  - 对外可见行为保持不变；新增/调整仅限命名层、文档层与治理门禁层。
  - 通过 alias、映射表与回滚说明降低批量重命名风险。
