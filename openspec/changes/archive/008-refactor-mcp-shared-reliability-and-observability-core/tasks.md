## 1. 共享核心抽取（内部封装）

- [x] 1.1 新建 `mcp/internal/*` 共享核心模块，定义统一调用执行骨架（timeout/retry/backoff/fail-fast）
- [x] 1.2 在共享核心中收敛 MCP 事件模板与诊断映射逻辑，并提供 transport hook 接口
- [x] 1.3 增加边界约束检查，确保 `mcp/internal/*` 不被非 MCP 包引用

## 2. HTTP/STDIO 重构接入

- [x] 2.1 将 `mcp/http` 改造为接入共享核心，保留连接/心跳/transport 专属逻辑
- [x] 2.2 将 `mcp/stdio` 改造为接入共享核心，保留 pool/backpressure/transport 专属逻辑
- [x] 2.3 清理已被共享核心覆盖的重复实现，保持对外 API 与行为兼容

## 3. 契约测试与重复逻辑度量

- [x] 3.1 新增跨 transport 契约测试矩阵（retryable/non-retryable/timeout/backpressure/reconnect）
- [x] 3.2 增加重复逻辑统计脚本或检查步骤，输出重构前后对比与相对下降百分比
- [x] 3.3 将“重复逻辑下降比例达到阈值”纳入验收记录与变更产物

## 4. 文档与质量门禁对齐

- [x] 4.1 更新 `docs/mcp-runtime-profiles.md`，说明共享核心与 transport-specific 分层
- [x] 4.2 更新 `docs/runtime-module-boundaries.md` 与 README 中的架构说明，补充 `mcp/internal/*` 边界
- [x] 4.3 执行并通过 `go test ./...`、`golangci-lint`、相关边界/文档检查脚本
