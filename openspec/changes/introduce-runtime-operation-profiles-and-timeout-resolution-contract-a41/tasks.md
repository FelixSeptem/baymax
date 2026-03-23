## 1. Config Schema and Resolver Foundation

- [ ] 1.1 在 `runtime/config` 增加 `runtime.operation_profiles.*` 配置结构、默认值与 `env > file > default` 解析。
- [ ] 1.2 为 profile 集合（`legacy|interactive|background|batch`）和 timeout 字段补齐启动期 fail-fast 校验。
- [ ] 1.3 在热更新路径补齐 operation profile 非法更新回滚测试，确保原子回滚语义。
- [ ] 1.4 实现共享 timeout resolver，固定三层优先级 `profile -> domain -> request` 并输出来源标签。

## 2. Scheduler and Composer Wiring

- [ ] 2.1 在 `orchestration/composer` 托管请求入口增加 operation profile 入参校验与透传。
- [ ] 2.2 在 `orchestration/scheduler` 子任务路径接入共享 timeout resolver，移除路径内 ad-hoc timeout 解析。
- [ ] 2.3 实现父子预算收敛逻辑 `min(parent_remaining, child_resolved)`，覆盖 exhausted budget reject 分支。
- [ ] 2.4 在 snapshot/recovery/replay 路径持久化并恢复 timeout-resolution 元数据，保持 idempotent 语义。

## 3. Diagnostics and Query Surface

- [ ] 3.1 在 `runtime/diagnostics` 增加 additive 字段：`effective_operation_profile`、`timeout_resolution_source`、`timeout_resolution_trace`。
- [ ] 3.2 增加 timeout 收敛计数聚合字段：`timeout_parent_budget_clamp_total`、`timeout_parent_budget_reject_total`。
- [ ] 3.3 确保 QueryRuns/Task Board 相关输出可返回 timeout-resolution 摘要且兼容旧消费者（nullable/default）。

## 4. Contract Tests and Gate Integration

- [ ] 4.1 新增 integration 套件覆盖 operation profile 校验、三层优先级、父子预算夹紧与 exhausted reject。
- [ ] 4.2 新增 Run/Stream 等价与 memory/file parity contract 用例，验证 timeout-resolution 行为一致。
- [ ] 4.3 新增 replay idempotency 用例，验证 timeout-resolution 聚合不膨胀。
- [ ] 4.4 更新 `scripts/check-multi-agent-shared-contract.*` 与 `scripts/check-quality-gate.*`，纳入阻断执行。

## 5. Documentation and Acceptance

- [ ] 5.1 更新 `docs/runtime-config-diagnostics.md`，补充 operation profile 字段、默认值、校验与环境变量映射。
- [ ] 5.2 更新 `docs/mainline-contract-test-index.md` 与 `docs/development-roadmap.md`，补充 A41 契约映射与状态。
- [ ] 5.3 更新 `README.md` 与相关模块 README（`runtime/config`、`runtime/diagnostics`、`orchestration/composer`）的能力描述。
- [ ] 5.4 执行并记录最小验收命令：`go test ./...`、`go test -race ./...`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-quality-gate.ps1`。
