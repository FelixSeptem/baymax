## 1. Retriever SPI And Provider Paths

- [x] 1.1 在 `context/provider` 定义统一 Retriever SPI（请求/响应/错误归一模型）
- [x] 1.2 保持 `file` provider 兼容并接入新 SPI
- [x] 1.3 实现 `http` provider（HTTP + JSON 映射 + 鉴权头支持）
- [x] 1.4 实现 `rag` provider 可运行路径（基于 SPI，不绑定供应商 SDK）
- [x] 1.5 实现 `db` provider 可运行路径（基于 SPI，不绑定供应商 SDK）
- [x] 1.6 实现 `elasticsearch` provider 可运行路径（基于 SPI，不绑定供应商 SDK）

## 2. Config And Validation

- [x] 2.1 扩展 `runtime/config` 的 CA2 Stage2 provider 枚举：`file|http|rag|db|elasticsearch`
- [x] 2.2 增加 external retriever 配置字段（endpoint、auth、headers、request/response JSON 映射）
- [x] 2.3 补齐启动与热更新校验（provider 枚举、endpoint 必填、映射格式合法）
- [x] 2.4 保持配置优先级 `env > file > default` 且与现有 manager 行为一致

## 3. Assembler And Diagnostics

- [x] 3.1 将 CA2 Stage2 调用统一切换到 SPI，保留现有 fail_fast/best_effort 语义
- [x] 3.2 为 Stage2 结果补齐 diagnostics 字段：`stage2_hit_count`、`stage2_source`、`stage2_reason`
- [x] 3.3 保持事件/诊断输出经过统一 redaction 管线（含鉴权相关字段）

## 4. Tests

- [x] 4.1 新增 provider 单元测试（file/http/rag/db/elasticsearch）
- [x] 4.2 新增配置校验测试（无效 provider、缺失 endpoint、非法 JSON 映射）
- [x] 4.3 新增 assembler 回归测试（fail_fast 与 best_effort 下 Stage2 行为一致）
- [x] 4.4 新增最小集成测试（mock HTTP retriever，不依赖真实外部服务）
- [x] 4.5 执行 `go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`

## 5. Documentation Alignment

- [x] 5.1 更新 `README.md`（CA2 provider 能力与外部检索接入说明）
- [x] 5.2 更新 `docs/runtime-config-diagnostics.md`（external retriever 配置、诊断字段说明）
- [x] 5.3 更新 `docs/context-assembler-phased-plan.md`（CA2 边界变更与 CA3 前置关系）
- [x] 5.4 更新 `docs/development-roadmap.md`（Knowledge 条目进展与下一阶段 TODO）
