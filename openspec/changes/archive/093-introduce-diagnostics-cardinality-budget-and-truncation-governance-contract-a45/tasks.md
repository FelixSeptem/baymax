## 1. Runtime Config Cardinality Governance

- [x] 1.1 在 `runtime/config` 增加 `diagnostics.cardinality.*` 配置结构、默认值与 `env > file > default` 映射。
- [x] 1.2 补齐 `diagnostics.cardinality.overflow_policy` 与预算阈值（map/list/string）的 startup 校验逻辑。
- [x] 1.3 补齐热更新 fail-fast + 原子回滚测试，覆盖非法 enum、非法阈值与布尔解析异常。

## 2. Diagnostics Budget and Truncation Implementation

- [x] 2.1 在 `runtime/diagnostics` 实现 map/list/string 预算检查入口，并在持久化前执行预算治理。
- [x] 2.2 实现 deterministic 截断策略（map key 排序后截断、list 前 N 保序、string 按字节并保持 UTF-8 边界）。
- [x] 2.3 实现 `truncate_and_record` 与 `fail_fast` 两种 overflow 策略，保证同输入同配置结果一致。

## 3. Observability and Compatibility

- [x] 3.1 在 `runtime/diagnostics` 新增 cardinality additive 字段与有界摘要字段（`additive + nullable + default`）。
- [x] 3.2 补齐 QueryRuns/相关查询输出映射，确保字段可查询且不引入高基数自由文本。
- [x] 3.3 补齐 replay idempotency 回归测试，确保重复回放不膨胀 cardinality 逻辑聚合。

## 4. Contract Tests and Quality Gate Wiring

- [x] 4.1 在 `integration` 增加 cardinality contract suites（budget、overflow policy、Run/Stream 等价、memory/file parity）。
- [x] 4.2 在 `integration` 增加 deterministic truncation 与 truncation summary 稳定性测试。
- [x] 4.3 更新 `scripts/check-quality-gate.sh` 与 `scripts/check-quality-gate.ps1`，将 diagnostics-cardinality suites 作为阻断步骤接入并保持 shell/PowerShell parity。

## 5. Documentation and Acceptance

- [x] 5.1 更新 `docs/runtime-config-diagnostics.md`，补齐 `diagnostics.cardinality.*` 字段、默认值、校验规则与诊断字段说明。
- [x] 5.2 更新 `docs/mainline-contract-test-index.md`、`docs/development-roadmap.md` 与 `README.md` 的 A45 状态和门禁映射。
- [x] 5.3 执行并记录最小验收命令：`go test ./...`、`go test -race ./...`、`pwsh -File scripts/check-docs-consistency.ps1`、`pwsh -File scripts/check-quality-gate.ps1`。
- [x] 5.4 修复并验证 `strict` 模式下 `govulncheck` 可用性（漏洞源可达、代理配置正确）与 Go module cache 写权限，确保 `pwsh -File scripts/check-quality-gate.ps1` 在 strict 模式全绿。
