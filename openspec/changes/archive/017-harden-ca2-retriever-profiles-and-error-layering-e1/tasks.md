## 1. Profile Templates And Config Validation

- [x] 1.1 在 `runtime/config` 为 CA2 Stage2 external 增加 `profile` 字段与默认模板集合（`http_generic`、`ragflow_like`、`graphrag_like`、`elasticsearch_like`）
- [x] 1.2 实现 profile 默认值合并逻辑（profile defaults -> 显式配置覆盖）并保持 `env > file > default`
- [x] 1.3 扩展配置校验：非法 profile、缺失关键字段、映射冲突按 fail-fast 报错
- [x] 1.4 增加 external retriever 预检查库接口，统一输出 warning/error findings

## 2. Retriever Error Layering And Diagnostics

- [x] 2.1 在 `context/provider` 引入 Stage2 错误分层（transport/protocol/semantic）与 reason code 映射
- [x] 2.2 在 `context/assembler` 保持 fail_fast/best_effort 行为不变，并写入 `stage2_reason_code`、`stage2_error_layer`、`stage2_profile`
- [x] 2.3 在 `runtime/diagnostics` 与 `observability/event` 扩展新增字段写入与读取，保持旧字段兼容
- [x] 2.4 确认 redaction 管线覆盖新增字段与 payload 路径

## 3. Tests And Quality Gates

- [x] 3.1 新增配置单测：profile 合并、优先级、预检查 warning/error 行为
- [x] 3.2 新增 provider/assembler 单测：错误分层、reason code、stage policy 不回归
- [x] 3.3 新增最小集成测试：profile 驱动 external retriever + diagnostics 字段验证（mock HTTP）
- [x] 3.4 执行 `go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`

## 4. Documentation Alignment

- [x] 4.1 更新 `README.md`（profile 配置、错误分层与诊断字段说明）
- [x] 4.2 更新 `docs/runtime-config-diagnostics.md`（预检查 API、warning/error 语义、新增字段）
- [x] 4.3 更新 `docs/v1-acceptance.md`（移除过时的 rag/db not-ready 描述，保持与实现一致）
- [x] 4.4 执行 docs 一致性检查并修复差异（含 roadmap/phase 相关引用）
