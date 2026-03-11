# Development Roadmap

更新时间：2026-03-11

## 目标

在当前 v1 基线能力上，进入可发布、可运营、可扩展的工程化阶段，重点提升：
- 稳定性（错误恢复、兼容性、回归防护）
- 可运维性（配置、观测、调试工具）
- 可扩展性（模型/工具/MCP/技能生态）

## 阶段规划

## Phase R1（2-4 周）稳定化与发布准备

### 进展（2026-03-11）
- [x] `upgrade-openai-native-stream-mapping`：完成 OpenAI 原生流式映射、fail-fast 终止语义、complete-tool-call-only 事件发射。
- [x] 增加 streaming golden tests 与回归用例（顺序、错误分类、Run/Stream 语义一致性）。
- [x] 引入 `golangci-lint` 配置与 CI 工作流（`go test` + `golangci-lint`）。
- [x] MCP HTTP/stdio 统一配置对象与默认值文档化（由 `harden-mcp-runtime-reliability-profiles` 完成）。

### 目标
- 冻结 v1 API 草案并补齐回归测试矩阵。
- 清理实现中的 compatibility-only 路径，降低行为歧义。

### 交付项
- 为 `model/openai` 补全原生流式映射（替换当前兼容实现）。
- 为 MCP HTTP/stdio 增加统一配置对象与文档化默认值。
- 增加 golden tests（事件序列、错误分类、tool feedback 合并）。
- 引入 lint + test + benchmark 的 CI。

### 验收标准
- `go test ./...` 稳定通过。
- 关键路径（run/tool/mcp/stream）覆盖率达到团队约定阈值。
- 事件/日志/trace 在一条运行内可 100% 关联。

## Phase R2（4-6 周）生产可运维能力

### 进展（2026-03-11）
- [x] `add-runtime-config-and-diagnostics-api-with-hot-reload`：完成 Viper 配置加载（YAML + Env + Default）、原子热更新、回滚语义与库级诊断 API。

### 目标
- 支持线上部署场景下的调优与排障。
- 建立并发与异步执行机制的可调优与可观测闭环。

### 交付项
- 配置层（环境变量 + 文件）和热更新策略。
- 观测增强：采样率、日志级别、慢调用阈值告警字段。
- MCP 健康检查与自愈策略（指数退避、熔断窗口、重连上限）。
- 运行诊断 API（导出最近 N 次 run/MCP 调用摘要，库接口）。
- 并发调度策略（队列/背压/取消传播）与异步通讯机制收敛。
- 交付 R2 批次示例：`01-chat-minimal`、`02-tool-loop-basic`、`03-mcp-mixed-call`、`04-streaming-interrupt`（附 TODO 演进位）。

### 验收标准
- 在故障注入测试中，MCP 间歇性错误可自动恢复。
- 关键指标可通过单一 dashboard 观测（延迟、错误率、重试率）。

## Phase R3（6-8 周）生态扩展与开发者体验

### 目标
- 降低新接入成本，增强外部集成能力。

### 交付项
- 模型适配接口文档与示例（多 provider）。
- Tool SDK 指南（schema、错误语义、幂等建议）。
- Skill 语义触发升级（可插拔检索/打分器）。
- 提供最小 CLI 示例（本地调试和回放）。
- 交付 R3 高阶示例：`05-parallel-tools-fanout`、`06-async-job-progress`、`07-multi-agent-async-channel`。

### 验收标准
- 新工具接入时间显著缩短（按团队 KPI 评估）。
- 外部团队可根据文档独立完成接入。

## Phase R4（长期）平台化能力（非 v1）

### 方向
- 持久化恢复与跨会话编排
- 多租户与权限治理
- 审计与合规流水线
- 分布式执行与弹性调度

## 技术债清单（当前建议优先）

- 清理仓库中的临时/备份产物与目录规范化。
- 收敛 `mcp/http` 与 `mcp/stdio` 中重复的重试/事件逻辑到共享组件。
- 为 `skill/loader` 的语义匹配引入可测试的评分接口。
- 为 runner 添加更多压力测试（高并发工具调用 + 取消风暴场景）。

## 性能与并发安全基线

- 性能回归采用相对提升百分比规则，详见 `docs/performance-policy.md`。
- 并发安全为强制门禁：`go test -race ./...` + goroutine 泄漏检查。

## 发布节奏建议

- 每周：1 次内部预发布（含 benchmark 回归对比）
- 每双周：1 次稳定 tag（附变更日志与风险说明）
- 每月：1 次架构评审（评估是否进入下一 phase）
