## 1. Config and Contract

- [x] 1.1 在 `runtime/config` 增加 CA3 compaction 配置：`mode`、semantic timeout、evidence retention（keywords + recent window）
- [x] 1.2 增加配置校验与热更新校验（非法配置 fail-fast）
- [x] 1.3 补充配置默认值与文档映射（`truncate` 默认）

## 2. CA3 Compaction SPI and Strategy

- [x] 2.1 在 `context/assembler` 引入包内 `Compactor SPI` 与策略路由器
- [x] 2.2 迁移现有截断逻辑到 `truncate` compactor，保证默认行为回归通过
- [x] 2.3 实现 `semantic` compactor，并通过现有 LLM client 路径执行语义压缩
- [x] 2.4 接入 semantic 超时与错误分类，保持状态一致性
- [x] 2.5 实现语义失败处理：`best_effort` 回退 truncate，`fail_fast` 终止

## 3. Evidence Retention and Diagnostics

- [x] 3.1 在 prune 候选筛选中加入 evidence retention 规则（关键词 + 最近窗口）
- [x] 3.2 新增 CA3 compaction 诊断字段：`ca3_compaction_mode`、`ca3_compaction_fallback`、`ca3_compaction_retained_evidence_count`
- [x] 3.3 确保 Run/Stream 对齐输出 compaction 语义字段

## 4. Tests and Quality Gate

- [x] 4.1 增加契约测试：semantic 启用路径、truncate 默认路径
- [x] 4.2 增加契约测试：`best_effort` fallback 与 `fail_fast` 终止语义
- [x] 4.3 增加契约测试：evidence retention 在 danger/emergency 的保护行为
- [x] 4.4 增加契约测试：Run/Stream 语义一致（mode/fallback/retained count）
- [x] 4.5 执行并通过 `go test ./...`
- [x] 4.6 执行并通过 `go test -race ./...`
- [x] 4.7 执行并通过 `golangci-lint run --config .golangci.yml`

## 5. Docs and Roadmap Sync

- [x] 5.1 更新 `README.md`（CA3 compaction 策略与最小配置）
- [x] 5.2 更新 `docs/runtime-config-diagnostics.md`（配置字段、诊断字段、回退语义）
- [x] 5.3 更新 `docs/context-assembler-phased-plan.md`（CA3 语义压缩策略与边界）
- [x] 5.4 更新 `docs/v1-acceptance.md` 与 `docs/mainline-contract-test-index.md`
- [x] 5.5 更新 `docs/development-roadmap.md`：登记“semantic 质量增强（评分/模板化/embedding 接口）”的后续 TODO
