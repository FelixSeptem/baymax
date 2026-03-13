## 1. Example Scaffolding

- [x] 1.1 新增 `examples/05-parallel-tools-fanout` 最小可运行示例（含 `main.go` 与 `TODO.md`）
- [x] 1.2 新增 `examples/06-async-job-progress` 最小可运行示例（含结构化 event 输出与 `TODO.md`）
- [x] 1.3 新增 `examples/07-multi-agent-async-channel` 最小可运行示例（含结构化 event 输出与 `TODO.md`）
- [x] 1.4 新增 `examples/08-multi-agent-network-bridge` 最小可运行示例（含 HTTP + JSON-RPC 2.0 通信、结构化 event 输出与 `TODO.md`）

## 2. Runtime Manager And Observability Alignment

- [x] 2.1 为 05-08 统一接入 `runtime/config.Manager`（配置加载与诊断入口一致）
- [x] 2.2 在 06-08 输出结构化事件，包含 run/stage/progress 等关键关联字段
- [x] 2.3 为 08 实现 HTTP 承载的 JSON-RPC 2.0 请求/响应与错误语义（参考 MCP 协议消息结构）
- [x] 2.4 确认示例仅复用现有核心能力，不引入 core/runtime 行为变更

## 3. Documentation Sync

- [x] 3.1 更新 `README.md`，新增按 PocketFlow Pattern 的示例导航索引表
- [x] 3.2 更新 `docs/examples-expansion-plan.md`，同步 05-08 实际落地与拆分说明
- [x] 3.3 更新 `docs/development-roadmap.md`，标注 R3 示例批次进展

## 4. Validation

- [x] 4.1 执行 `go test ./...`，确保示例目录可编译
- [x] 4.2 手动运行 05-08 示例入口，验证“可运行 + 结构化输出”目标
