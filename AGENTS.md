# AGENTS.md

本文档面向在 `baymax` 仓库内工作的 AI coding agent 与贡献者，目标是保证改动可落地、可回归、可评审。

## 1. 项目概览

Baymax 是一个 `library-first` 的 Go Agent Loop 运行时，核心能力包括：

- `core/runner` 的 `Run` / `Stream` 双路径状态机
- 本地工具调度与 MCP 双传输（`mcp/http`、`mcp/stdio`）
- 多 Provider 模型适配（`openai`、`anthropic`、`gemini`）
- Context Assembler（CA1-CA4）
- 运行时配置/诊断（热更新 + 回滚）
- 结构化可观测性（timeline + OTel + runtime recorder）
- 技能加载与触发评分

权威文档入口：

- `README.md`
- `docs/runtime-config-diagnostics.md`
- `docs/runtime-module-boundaries.md`
- `docs/development-roadmap.md`
- `CONTRIBUTING.md`

## 2. 仓库结构速览

- `core/runner`：主循环编排与终止语义
- `tool/local`：本地工具注册、调度、策略
- `mcp/http`, `mcp/stdio`：MCP 传输实现
- `mcp/profile`, `mcp/retry`, `mcp/diag`：MCP 语义域
- `mcp/internal/*`：MCP 内部共享组件（限制外部引用）
- `model/*`：Provider SDK 适配与能力探测
- `context/*`：上下文装配、journal、guard、CA 策略
- `runtime/config`：统一配置加载/校验/热更新
- `runtime/diagnostics`：统一诊断模型与查询
- `observability/event`：事件发射与 `RuntimeRecorder`
- `skill/loader`：技能发现、评分、bundle 组装
- `integration`：E2E 与 benchmark
- `openspec`：spec-driven 变更工件
- `scripts`：本地/CI 质量门禁脚本

## 3. 架构硬约束（必须遵守）

1. `runtime/*` 禁止依赖 `mcp/http` 或 `mcp/stdio`
2. 非 `mcp/*` 包禁止依赖 `mcp/internal/*`
3. `context/*` 禁止直接引入 Provider 官方 SDK
4. Provider 协议细节必须落在 `model/<provider>`
5. 诊断写入走 `observability/event.RuntimeRecorder` 单写入口
6. 配置优先级固定为 `env > file > default`
7. 非法配置与非法热更新必须 fail-fast，且原子回滚

若改动会触碰上述边界，先更新 OpenSpec 设计与相关文档，再改代码。

## 4. OpenSpec 工作流（默认优先）

涉及行为/配置/契约变化时，按 OpenSpec 执行：

1. 先看活跃变更：`openspec list --json`
2. 在变更目录实施：`openspec/changes/<change-name>/`
3. 对齐工件：`proposal.md`、`design.md`、`tasks.md`、`specs/*/spec.md`
4. 按 `tasks.md` 顺序实现并勾选
5. 保持代码、测试、文档与 spec delta 同步
6. 完成后按项目约定归档到 `openspec/changes/archive/`

纯内部重构且无外部行为变化时，可直接提交补丁，但仍需最小回归测试。

## 5. 开发与验证命令

### 环境准备

```bash
go mod tidy
```

### PR 前最低验证（必跑）

```bash
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml
```

### 质量门禁脚本

Linux/macOS：

```bash
bash scripts/check-quality-gate.sh
bash scripts/check-runtime-boundaries.sh
bash scripts/check-contribution-template.sh <github_event_path>
bash scripts/check-diagnostics-replay-contract.sh
bash scripts/check-security-policy-contract.sh
bash scripts/check-security-event-contract.sh
bash scripts/check-security-delivery-contract.sh
```

Windows PowerShell：

```powershell
pwsh -File scripts/check-quality-gate.ps1
pwsh -File scripts/check-docs-consistency.ps1
pwsh -File scripts/check-contribution-template.ps1 <github_event_path>
pwsh -File scripts/check-diagnostics-replay-contract.ps1
pwsh -File scripts/check-security-policy-contract.ps1
pwsh -File scripts/check-security-event-contract.ps1
pwsh -File scripts/check-security-delivery-contract.ps1
```

### Benchmark smoke（与 CI 对齐）

```bash
go test ./integration -run ^$ -bench Benchmark -benchtime=50ms
```

## 6. 测试策略要求

行为变更必须满足：

1. 受影响包补充或更新单测
2. 跨模块链路补 integration 用例
3. Run/Stream 语义保持一致（如适用）
4. 并发/取消相关改动至少跑 `go test -race ./...`
5. 性能敏感路径补 benchmark 或更新基线

没有测试的行为改动默认不可合入，除非在 PR 明确给出理由。

## 7. 文档同步要求

同一 PR 内同步更新文档，禁止“代码先行、文档滞后”：

- 配置字段/默认值/校验变化：同步 `runtime/config` 与 `docs/runtime-config-diagnostics.md`
- 诊断字段或语义变化：同步 `runtime/diagnostics` 与文档字段说明
- 模块职责边界变化：同步 `docs/runtime-module-boundaries.md`
- 用户可见用法变化：同步 `README.md` / `examples`
- 迁移影响：在 `CHANGELOG.md` 或 PR 描述写明

## 8. PR 交付标准

严格按 `.github/pull_request_template.md` 填写必填项：

- Summary
- Changes
- Validation
- Documentation
- Impact

要求：

1. 保持变更聚焦，避免无关 diff
2. 明确列出实际执行过的验证命令
3. 说明风险、回滚点与迁移影响（如有）

## 9. 安全与可靠性护栏

- 禁止提交密钥、令牌、敏感配置
- 不得破坏脱敏链路（`runtime/diagnostics`、`observability/event`、`context/assembler`）
- 安全治理相关改动必须跑对应 contract gate
- 需要保持 deny 决策语义不被回调投递失败所改变（按当前策略约束）

## 10. Agent 执行风格

1. 先读最小必要上下文，再动手实现
2. 优先小步、可审查补丁
3. 在 PR 记录关键假设与风险点
4. 若本地无法完成某项验证，要明确写出“未执行项 + 原因”
5. 不得静默绕过架构边界或 spec 约束

不确定时，优先契约稳定性与可回归性，而不是捷径实现。
