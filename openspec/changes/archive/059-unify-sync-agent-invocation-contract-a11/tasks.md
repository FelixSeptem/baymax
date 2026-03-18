## 1. Shared Sync Invocation Abstraction

- [x] 1.1 新增 `orchestration/invoke` 包并定义最小同步调用接口与结果模型。
- [x] 1.2 实现统一 `Submit -> WaitResult -> terminal normalize` 调用路径。
- [x] 1.3 实现默认 `poll_interval`（20ms）与可选覆盖逻辑。
- [x] 1.4 实现 `context` 取消/超时优先语义并补边界单测。
- [x] 1.5 实现错误分层归一与 retryable 提示逻辑。

## 2. Orchestration Integration Refactor

- [x] 2.1 将 `orchestration/scheduler` A2A 适配路径切换到共享同步调用抽象。
- [x] 2.2 将 `orchestration/composer` `ChildTargetA2A` 路径切换到共享同步调用抽象。
- [x] 2.3 将 `orchestration/workflow` `StepKindA2A` 路径切换到共享同步调用抽象。
- [x] 2.4 将 `orchestration/teams` `TaskTargetRemote` 路径切换到共享同步调用抽象。
- [x] 2.5 清理模块内重复同步调用拼装逻辑并保持行为兼容。

## 3. Contract & Regression Tests

- [x] 3.1 为 `orchestration/invoke` 增加单测：成功、超时、取消、transport/protocol/semantic 错误。
- [x] 3.2 增加 integration 套件覆盖 workflow/teams/composer/scheduler 的同步语义一致性。
- [x] 3.3 增加 Run/Stream 等价测试，覆盖同步远程调用聚合字段一致性。
- [x] 3.4 增加 scheduler 路径 `canceled` 终态映射与 retryable 语义回归测试。

## 4. Gate and Documentation Alignment

- [x] 4.1 将 A11 同步调用契约测试并入 `check-multi-agent-shared-contract.sh/.ps1` 阻断路径。
- [x] 4.2 更新 `docs/mainline-contract-test-index.md` 增加 A11 测试映射。
- [x] 4.3 更新 `README.md` 增加统一同步调用推荐用法与最小示例。
- [x] 4.4 更新 `docs/runtime-config-diagnostics.md` 说明同步语义（无新增破坏性配置）。
- [x] 4.5 按需更新 `docs/runtime-module-boundaries.md` 声明 `orchestration/invoke` 所属边界。

## 5. Validation and Closure

- [x] 5.1 执行 `go test ./...`。
- [x] 5.2 执行 `$env:CGO_ENABLED='1'; go test -race ./...`。
- [x] 5.3 执行 `golangci-lint run --config .golangci.yml`。
- [x] 5.4 执行 `pwsh -File scripts/check-docs-consistency.ps1`。
- [x] 5.5 执行 `pwsh -File scripts/check-multi-agent-shared-contract.ps1`。
