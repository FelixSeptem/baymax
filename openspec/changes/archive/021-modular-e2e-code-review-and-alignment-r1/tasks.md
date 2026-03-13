## 1. Review Matrix and Findings Baseline

- [x] 1.1 建立模块评审清单（`core/context/model/runtime/observability`）并固化检查项（职责、错误语义、并发安全、可观测一致性）
- [x] 1.2 建立主干流程串联清单（`Run`、`Stream`、`tool-loop`、`CA2 stage2`、`CA3 pressure/recovery`）并定义期望语义
- [x] 1.3 输出 P0/P1/P2 问题清单并绑定到对应文件与测试缺口

## 2. Full-Severity Fix Convergence (P0+P1+P2)

- [x] 2.1 按清单完成 P0 问题修复，确保 fail-fast 与错误分类语义不回退
- [x] 2.2 按清单完成 P1 问题修复，确保模块边界与主链路行为一致
- [x] 2.3 按清单完成 P2 问题修复，消除可维护性与语义漂移隐患
- [x] 2.4 复核所有已识别问题状态为 closed（本提案不遗留 P0/P1/P2）

## 3. Mainline Contract Test Coverage

- [x] 3.1 为 `Run` 与 `Stream` 增补/修正契约测试，覆盖成功与失败终止语义
- [x] 3.2 为 `tool-loop`、`CA2 stage2`、`CA3 pressure/recovery` 增补/修正契约测试，覆盖正常与降级路径
- [x] 3.3 建立主干流程到测试用例的映射索引，确保覆盖可追踪
- [x] 3.4 执行 `go test ./...` 与 `go test -race ./...`，修复回归直至稳定通过

## 4. Repository Hygiene and Guardrails

- [x] 4.1 清理仓库中的临时/备份产物（含 `*.go.<random>` 类文件）
- [x] 4.2 在质量门禁流程中增加仓库卫生检查，防止临时产物回流
- [x] 4.3 复核脚本保留清单，移除不再需要的脚本并保持 CI 语义一致

## 5. Documentation and Consistency

- [x] 5.1 同步更新 `README.md` 与 `docs/` 下受影响文档（边界、质量门禁、主干流程契约）
- [x] 5.2 执行文档一致性检查并修复不一致项
- [x] 5.3 在 change artifacts 中补充“评审发现 -> 修复 -> 测试覆盖 -> 文档同步”闭环说明

## 6. Final Quality Gate

- [x] 6.1 执行 `golangci-lint run --config .golangci.yml` 并修复问题
- [x] 6.2 执行 `govulncheck`（strict）并确认通过
- [x] 6.3 汇总验收证据（测试、lint、安全扫描、文档一致性）并准备归档
