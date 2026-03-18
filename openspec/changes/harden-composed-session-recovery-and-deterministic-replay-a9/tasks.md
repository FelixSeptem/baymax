## 1. Recovery 配置与模型

- [ ] 1.1 在 `runtime/config` 新增 recovery 配置域并设置默认 `recovery.enabled=false`。
- [ ] 1.2 增加 recovery 配置校验（冲突策略仅允许 `fail_fast`）及热更新回滚测试。
- [ ] 1.3 定义恢复快照模型（run/workflow/scheduler/a2a/replay cursor）与版本字段。

## 2. RecoveryStore 抽象与后端

- [ ] 2.1 实现 `RecoveryStore` 抽象接口并完成 `memory` 后端。
- [ ] 2.2 完成 `file` 后端（原子写入、加载校验、损坏快照错误分类）。
- [ ] 2.3 增加存储层单测（读写一致性、重复加载、损坏文件 fail-fast）。

## 3. Composer 恢复编排

- [ ] 3.1 在 composer 路径增加 resume/recover 入口与恢复状态机（load -> validate -> reconcile -> resume）。
- [ ] 3.2 打通 workflow checkpoint 恢复与 scheduler state 恢复的顺序编排。
- [ ] 3.3 将 A2A in-flight 状态纳入恢复收敛并保持 task correlation 映射稳定。
- [ ] 3.4 实现恢复冲突 `fail_fast` 终止路径与标准错误分类输出。

## 4. 重放幂等与语义一致

- [ ] 4.1 收敛恢复重放下 scheduler terminal commit 幂等语义（避免重复计数膨胀）。
- [ ] 4.2 收敛恢复重放下 workflow 与 A2A 终态收敛语义（避免重复副作用）。
- [ ] 4.3 增加 Run/Stream 恢复路径语义等价校验（终态类别 + 聚合字段）。

## 5. 观测与边界治理

- [ ] 5.1 增加 recovery/replay 事件与 run 摘要 additive 字段映射，保持 single-writer。
- [ ] 5.2 增加恢复相关 timeline reason 与必填关联字段断言。
- [ ] 5.3 更新边界检查，确保 recovery 逻辑不直接写 `runtime/diagnostics`。

## 6. 契约测试与门禁集成

- [ ] 6.1 新增 integration 套件：跨会话恢复成功、重复重放幂等、冲突 fail-fast。
- [ ] 6.2 新增/更新 contributioncheck 快照用例，覆盖 recovery 契约字段与 reason。
- [ ] 6.3 将 recovery 套件并入 `scripts/check-multi-agent-shared-contract.sh/.ps1` 阻断路径。
- [ ] 6.4 更新 `docs/mainline-contract-test-index.md` 对应 A9 测试映射条目。

## 7. 文档与发布口径

- [ ] 7.1 更新 `README.md` 与 `docs/runtime-config-diagnostics.md`（默认关闭、启用方式、冲突策略）。
- [ ] 7.2 更新 `docs/runtime-module-boundaries.md`、`docs/v1-acceptance.md`、`docs/development-roadmap.md`。
- [ ] 7.3 执行验证：`go test ./...`、`$env:CGO_ENABLED='1'; go test -race ./...`、`golangci-lint run --config .golangci.yml`、`pwsh -File scripts/check-multi-agent-shared-contract.ps1`。
