## 1. 单写入与幂等治理（P0）

- [x] 1.1 盘点 `core/runner`、`skill/loader`、`observability/event/runtime_recorder` 的诊断写入入口，确定唯一落库路径并移除另一侧落库逻辑
- [x] 1.2 在诊断写入模型中引入 run/skill 幂等键生成规则（稳定且可复现）并完成写入层去重
- [x] 1.3 为重试、重放、并发重复提交场景补充单元测试，验证“同语义仅一条逻辑记录”

## 2. 诊断契约加固（P1）

- [x] 2.1 收敛 run/skill 共享诊断字段、状态枚举与错误分类，更新领域模型与映射逻辑
- [x] 2.2 为 success/failure/warning/retry(replay) 场景补充 contract tests，覆盖 runner 与 skill loader 两条生产路径
- [x] 2.3 更新 diagnostics API 相关文档说明，明确单写入与幂等后的统计口径

## 3. 并发安全质量基线（P1）

- [x] 3.1 将 `go test ./...` 与 `go test -race ./...` 固化到标准校验流程（本地与 CI）
- [x] 3.2 新增诊断并发专项测试集（并发写入、重复事件重放、去重一致性）并纳入必跑
- [x] 3.3 更新 `go-quality-gate` 相关文档与脚本，确保并发安全检查失败即阻断合并
