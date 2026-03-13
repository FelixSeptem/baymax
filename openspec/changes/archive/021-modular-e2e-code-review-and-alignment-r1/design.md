## Context

当前仓库已经完成多批能力落地，核心路径覆盖 `runner`、`context assembler`、`multi-provider model`、`runtime diagnostics` 与 `action timeline`。但连续迭代后存在典型工程风险：
- 模块边界在局部修复中被弱化（尤其 context 与 model/provider 职责交叉点）。
- Run/Stream、tool-loop、CA2/CA3 等主干流程的语义一致性依赖分散测试，覆盖矩阵不够显式。
- 仓库中出现临时备份产物与文档漂移风险，影响维护效率与审阅准确性。

本提案定位为“收敛型治理变更”：不扩展新产品能力，聚焦代码质量、职责边界、端到端契约与文档一致性。

## Goals / Non-Goals

**Goals:**
- 建立模块化 code review 与端到端串联 review 的统一执行框架，并产出可落地问题清单。
- 在同一提案内一次性收敛 `P0 + P1 + P2`，不遗留已识别高优先级问题。
- 将主干流程契约测试覆盖固化为质量门禁要求，确保 Run/Stream 与关键子链路语义一致。
- 清理仓库临时/备份文件并补充防回流约束。
- 同步 README 与 docs，确保实现、门禁、文档三者一致。

**Non-Goals:**
- 不引入新的业务特性（如新增协议、新增编排能力、全新 provider）。
- 不重构整体架构层级，仅在现有边界内修复职责漂移与行为不一致。
- 不新增 CLI 产品面功能。

## Decisions

### 1) 采用“双视角评审”：模块横向 + 主干链路纵向
- 决策：评审过程分为两条并行清单。
  - 模块横向：`core`、`context`、`model`、`runtime`、`observability`。
  - 主干纵向：`Run`、`Stream`、`tool-loop`、`CA2 Stage2`、`CA3 pressure/recovery`。
- 理由：仅做模块 review 容易漏掉串联语义；仅做链路 review 又无法定位职责根因。
- 备选：只做全量 lint/test。
  - 放弃原因：无法系统发现语义/边界漂移。

### 2) 风险分级只用于排队，不用于拆批
- 决策：保留 P0/P1/P2 标注以便排序，但本提案内必须全部闭环。
- 理由：用户已确认一次性收敛，避免“已知问题滚动遗留”。
- 备选：P0 先做、P1/P2 后续提案。
  - 放弃原因：会延长漂移窗口，增加后续返工。

### 3) 契约测试以“主干流程覆盖率”作为硬门禁
- 决策：将主干流程契约测试纳入质量门禁，要求每条主干链路至少一个正向场景 + 一个异常/降级场景。
- 理由：本提案的目标是保障语义稳定，契约测试比单点单元测试更能反映真实回归风险。
- 备选：仅要求新增单元测试。
  - 放弃原因：难覆盖跨模块协同语义。

### 4) 仓库卫生纳入同一变更
- 决策：清理临时备份产物并在门禁中增加仓库卫生检查（例如拒绝 `*.go.<random>` 这类非源码产物）。
- 理由：此类文件会误导 review 与检索，属于质量治理的一部分。
- 备选：仅人工清理一次。
  - 放弃原因：容易回流且不可持续。

### 5) 文档同步作为完成定义的一部分
- 决策：实现修复后必须同步 README 与 docs 受影响页面；未同步视为未完成。
- 理由：当前项目以文档驱动提案流程运行，文档漂移会直接影响后续变更质量。
- 备选：文档延后补。
  - 放弃原因：容易遗漏并造成认知偏差。

## Risks / Trade-offs

- [Risk] 一次收敛 P0/P1/P2 会提高单次改动规模与审阅成本  
  → Mitigation: 先固定 review matrix，再按 matrix 提交修复与测试，确保每项可验证。
- [Risk] 主干契约测试扩容会增加 CI 时间  
  → Mitigation: 保持“主干最小充分集”，避免重复场景；必要时区分快速集与全量集。
- [Risk] 清理临时文件可能误删尚在使用的调试产物  
  → Mitigation: 仅清理明确命名模式的备份/随机后缀文件，并在 tasks 中加入复核步骤。
- [Risk] 文档同步范围大，易漏项  
  → Mitigation: 在 tasks 中显式列出 README + docs 清单核对步骤，并执行一致性脚本。

## Migration Plan

1. 建立本次评审矩阵（模块 + 串联）并产出问题清单（含 P0/P1/P2）。
2. 按矩阵顺序完成修复，确保同一问题的代码、测试、文档一起提交。
3. 补齐主干流程契约测试与仓库卫生检查，接入现有质量门禁脚本。
4. 执行 `go test ./...`、`go test -race ./...`、`golangci-lint`、`govulncheck`。
5. 更新 README + docs，并通过一致性检查后再进入归档。

## Open Questions

- 本提案不新增问题优先级等级（沿用 P0/P1/P2），后续是否需要引入“架构债”标签作为补充维度，待下一批治理提案再评估。

## Implementation Closure

- 评审发现：
  - P0：仓库临时备份文件回流风险（`*.go.<random>`）。
  - P1：CA3 token counter 在 provider fallback 选路后的注入契约缺少覆盖。
  - P1：provider token count 归一化语义缺少回归测试（Gemini role 归一化、OpenAI unsupported）。
  - P2：主干流程到测试用例缺少集中索引，文档与门禁语义未完全对齐。
- 修复动作：
  - 清理备份文件并新增仓库卫生检查脚本（Linux/PowerShell）接入质量门禁与 CI。
  - 补充 runner 契约测试，验证 Run/Stream 在 fallback 后使用 selected provider 的 token counter。
  - 补充 provider 级 token count 回归测试。
  - 新增评审矩阵与主干测试索引文档，更新 README/docs 对齐当前实现。
- 验收结果：
  - `go test ./...` 通过。
  - `go test -race ./...` 通过。
  - `golangci-lint run --config .golangci.yml` 通过。
  - `govulncheck ./...`（strict）通过（无可达漏洞）。
  - 文档一致性检查通过（`scripts/check-docs-consistency.ps1`）。
