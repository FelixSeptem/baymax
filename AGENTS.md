# AGENTS.md

适用范围：`baymax` 仓库内所有 AI agent 与贡献者。
目标：保证改动可回归、可评审、可归档。

## 必须遵守

1. OpenSpec 优先（行为/配置/contract 变更）
- 先看：`openspec list --json`
- 在 `openspec/changes/<change-name>/` 内维护 `proposal.md`、`design.md`、`tasks.md`、`specs/*/spec.md`
- 按 `tasks.md` 实施并勾选，代码/测试/文档与 spec delta 同步提交

2. 提案归档必须执行 scripts（含重命名）
- 禁止手工改 `openspec/changes/archive` 目录名
- 归档示例：
```powershell
pwsh -File scripts/openspec-archive-seq.ps1 -ChangeName "introduce-realtime-event-protocol-and-interrupt-resume-contract-a68"
```
- 历史归档重排：
```powershell
pwsh -File scripts/openspec-archive-seq.ps1 -MigrateExisting
```

3. 代码变更后必须跑门禁
```bash
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml
```
```powershell
pwsh -File scripts/check-quality-gate.ps1
pwsh -File scripts/check-docs-consistency.ps1
```

4. 新增/修改 contract 必须补测试（强制）
- 受影响包单测（正向/负向/边界）
- 必要的 integration 用例
- 对应 contract/replay/gate 覆盖（fixture、drift、脚本）

5. 命名必须语义化（禁止提案标号渗透）
- 代码标识、文件名、文件路径中禁止使用提案标号（如 `a62`、`a63`、`axx`）作为命名。
- 统一使用语义化命名（按业务/能力/模块语义命名），避免编号驱动命名。
- 例外：`openspec/changes/*` 与 `openspec/changes/archive/*` 的提案目录按 OpenSpec 工作流可保留提案标号。

6. 提案必须声明 Example Impact Assessment（行为/配置/契约变更）
- 当提案涉及运行时行为、配置语义、诊断 schema 或 contract 预期变化时，`proposal/design/tasks` 必须显式声明示例影响评估。
- 声明值限定为三选一：`新增示例`、`修改示例`、`无需示例变更（附理由）`。
- 缺少声明或声明值不合法，视为提案不完整，禁止进入实施阶段。

## 架构硬约束（不可绕过）

- `runtime/*` 禁止依赖 `mcp/http` 或 `mcp/stdio`
- 非 `mcp/*` 包禁止依赖 `mcp/internal/*`
- `context/*` 禁止直接引入 Provider 官方 SDK
- Provider 协议与模型适配细节必须落在 `model/<provider>`
- 诊断写入必须走 `observability/event.RuntimeRecorder` 单写入口
- 配置优先级固定 `env > file > default`
- 非法配置与非法热更新必须 `fail-fast + 原子回滚`
- QueryRuns/诊断字段变更遵循 `additive + nullable + default`
- Run/Stream 对等场景必须保持语义等价，不引入平行终止/决策语义

如触碰上述约束，必须先更新 OpenSpec 与对应文档，再改代码。

## 文档路径（变更时必查）

- `README.md`：对外能力总览与快速入口
- `docs/development-roadmap.md`：提案优先级、范围边界、DoD
- `docs/runtime-config-diagnostics.md`：配置键、默认值、校验与诊断字段
- `docs/runtime-module-boundaries.md`：模块依赖边界与分层约束
- `docs/mainline-contract-test-index.md`：contract/replay/gate 对应关系
- `openspec/changes/archive/INDEX.md`：已归档提案索引
- `CONTRIBUTING.md`：提交与协作约定
- `.github/pull_request_template.md`：PR 必填交付项

## PR 交付

- 同一 PR 同步代码、测试、文档
- PR 必须写明：验证命令、风险点、回滚点、未执行项与原因
