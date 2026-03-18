## 1. Async Reporting Contract Foundation

- [x] 1.1 定义 `SubmitAsync` 与 `ReportSink` 最小接口及数据模型。
- [x] 1.2 实现内置 `channel sink` 与 `callback sink`。
- [x] 1.3 实现异步回报幂等键生成与去重逻辑。
- [x] 1.4 实现回报失败重试（指数退避 + 抖动）与终止条件。
- [x] 1.5 保证回报失败不改变业务终态语义并补单测。

## 2. A2A and Orchestration Integration

- [x] 2.1 在 `a2a` 增加异步提交入口并保留现有同步兼容路径。
- [x] 2.2 接入 composer 子任务异步回报汇聚路径。
- [x] 2.3 接入 scheduler 异步回报终态回填与去重路径。
- [x] 2.4 保证 workflow/teams 在需要时可消费异步回报结果而不强制阻塞等待。

## 3. Config, Diagnostics, and Timeline

- [x] 3.1 增加 `a2a.async_reporting.*` 配置域、默认值与 fail-fast 校验。
- [x] 3.2 扩展 run diagnostics additive 字段：total/failed/retry/dedup。
- [x] 3.3 增加 async reporting timeline reason taxonomy 与关联字段映射。
- [x] 3.4 保证新增字段遵循 `additive + nullable + default` 兼容窗口。

## 4. Contract Tests and Quality Gate

- [x] 4.1 增加异步回报契约测试：成功回报、回报失败重试、最终失败。
- [x] 4.2 增加幂等去重与 replay-idempotent 契约测试。
- [x] 4.3 增加 Run/Stream 等价测试（异步回报场景）。
- [x] 4.4 增加 recovery 场景一致性测试（回放不膨胀计数）。
- [x] 4.5 将异步回报套件并入 `check-multi-agent-shared-contract.sh/.ps1`。

## 5. Documentation and Validation

- [x] 5.1 更新 `README.md`：异步提交与回报最小示例。
- [x] 5.2 更新 `docs/runtime-config-diagnostics.md`：配置与诊断字段说明。
- [x] 5.3 更新 `docs/mainline-contract-test-index.md`：A12 测试映射。
- [x] 5.4 更新 `docs/development-roadmap.md`：A12 状态与后续顺序。
- [x] 5.5 执行 `go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-multi-agent-shared-contract.ps1`。
