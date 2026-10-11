## Context

现有 Documentation Impact gate 已检查九个文档领域，但没有一个跨 `proposal.md`、`design.md`、`tasks.md` 的字段表示提案作者是否接受公开语义变化必须同步维护文档的规则。治理 spec 目前只覆盖 roadmap 状态和 Example Impact Assessment。详见 proposal.md 的 Why。

## Goals / Non-Goals

**Goals:**

- 为三份 OpenSpec 工件建立同名、可解析的公开语义文档同步承诺。
- 对“有公开语义变化但没有维护文档证据”和“无变化但没有理由”提供确定性失败码。
- 让 PowerShell 与 shell gate 使用相同的字段、合法值和漂移判断。
- 保持现有 Documentation Impact 表格兼容，并将承诺作为其前置接受条件。

**Non-Goals:**

- 不自动推断所有代码是否改变公开语义；提案作者必须声明，gate 校验声明完整性和跨工件一致性。
- 不强制所有内部重构新增用户文档；无公开语义变化可以声明否，但必须给出具体理由。
- 不改变 runtime、provider、配置、诊断或 examples/agent-modes 行为。

## Decisions

1. **固定段落和字段。** 三份工件都必须有 `## Public Semantic Documentation Commitment`，并包含 `对外语义变化`、`文档同步状态`、`维护文档`、`维护 owner`、`验证证据` 五个字段。固定字段比自由文本更适合 shell/PowerShell parity。
2. **使用二值语义声明。** `对外语义变化` 只允许 `是` 或 `否`。`是` 要求同步状态为 `已同步`、维护文档不是 `—` 且包含路径和证据；每个路径必须存在且出现在当前 change 相对基准分支的 Git 差异中（包含未提交、未跟踪文件）。`否` 要求同步状态以“无需同步（附理由）”开头且理由足够具体。
3. **三份工件必须一致。** gate 比较三份工件的语义变化、同步状态、维护文档路径、owner 和验证证据；“是”场景还检查文档存在且随当前 change 修改。这样 design/tasks 不能弱化 proposal 的承诺。
4. **保留现有影响表。** 承诺不替代九领域 Documentation Impact Assessment。影响表继续表达每个文档领域的路径和验证，承诺负责确认作者接受同步规则。
5. **只检查 active changes。** 归档内容不被新 gate 重新解释；归档时任务和 verification 负责记录当时的承诺，避免历史提案重写。

## Risks / Trade-offs

- [Risk] 现有 active change 缺少新段落而导致门禁失败 → 当前没有 active change；后续新提案模板和 proposal instructions 会引导补齐，错误码提供明确修复方向。
- [Risk] 自由文本理由难以判断质量 → gate 只要求非空且超过最小长度，并保留人工 review；不尝试用关键词替代语义审查。
- [Risk] shell 与 PowerShell 解析边界不同 → 两个实现使用相同字段和固定前缀，并对合法/非法 fixture 做 parity 验证。

## Migration Plan

1. 修改治理规则、模板和双平台 gate。
2. 为本 change 的三份工件补齐承诺并运行 gate。
3. 后续 active change 必须在创建时写入承诺；归档历史不迁移。
4. 回滚时移除新 gate 检查和模板段落，不涉及持久化或运行时迁移。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`AGENTS.md`、`README.md`、`docs/documentation-style-guide.md`、`docs/development-roadmap.md`、`docs/guides/openspec-workflow.md`、`openspec/changes/archive/161-enforce-openspec-public-semantics-documentation-contract/specs/proposal-governance-automation-contract/spec.md`
- 维护 owner：governance maintainers
- 验证证据：双平台 documentation impact gate、docs consistency、OpenSpec strict validation

## Documentation Impact Assessment

| Area | Outcome | Affected paths | Owner | Verification |
| --- | --- | --- | --- | --- |
| architecture | 无需文档变更（附理由） | — | maintainer | no runtime architecture changes |
| components | 无需文档变更（附理由） | — | package owners | no component behavior changes |
| configuration | 无需文档变更（附理由） | — | runtime/config | no configuration changes |
| contract/API | 修改文档 | `AGENTS.md`, `openspec/changes/archive/161-enforce-openspec-public-semantics-documentation-contract/specs/proposal-governance-automation-contract/spec.md` | governance maintainers | OpenSpec strict validation |
| diagnostics | 无需文档变更（附理由） | — | observability | no diagnostics schema changes |
| examples | 无需文档变更（附理由） | — | example owners | Example Impact gate |
| CLI/integration | 修改文档 | `README.md`, `docs/development-roadmap.md` | release owners | docs consistency gate |
| best practices | 修改文档 | `docs/documentation-style-guide.md` | maintainer | documentation impact gate |
| roadmap | 修改文档 | `docs/development-roadmap.md` | release owner | roadmap status gate |
