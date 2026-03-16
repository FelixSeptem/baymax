## 1. Runtime Config for Skill Trigger Scoring

- [x] 1.1 在 `runtime/config` 新增 `skill.trigger_scoring` 配置结构（策略、阈值、tie-break、低置信度抑制、关键词权重）并设置默认值
- [x] 1.2 将新增字段接入 `env > file > default` 解析链路与热更新快照
- [x] 1.3 为新增枚举/范围/映射字段补齐 fail-fast 校验（启动与热更新一致）
- [x] 1.4 增加配置测试：默认值、YAML 覆盖、ENV 覆盖、非法输入失败路径

## 2. Skill Loader Scoring Core (Internal)

- [x] 2.1 在 `skill/loader` 抽象内部 scorer 接口，并保持仅包内/内部复用
- [x] 2.2 实现默认 lexical weighted-keyword scorer，替换当前内嵌评分逻辑
- [x] 2.3 实现同分规则 `highest-priority` 的确定性选择逻辑
- [x] 2.4 实现低置信度抑制默认开启行为，并支持通过 runtime 配置关闭
- [x] 2.5 为 embedding scorer 预留 internal 接入 TODO（不实现具体 embedding 行为）

## 3. Contract Tests and Regression Matrix

- [x] 3.1 新增/更新 `skill/loader` 合同测试：阈值命中、低置信度过滤、同分高优先级选择
- [x] 3.2 新增配置驱动测试：scoring 配置变更后 loader 行为同步变化
- [x] 3.3 增加稳定性测试：同输入多次执行结果一致（deterministic tie-break）
- [x] 3.4 补充 Run/Stream 语义不变性回归测试（确保仅触发选择行为变化，不破坏主干终止语义）

## 4. Diagnostics and Event Consistency

- [x] 4.1 审核 skill 事件/诊断字段，确保新增评分行为不破坏既有诊断契约
- [x] 4.2 如需新增最小字段，保持后向兼容并补齐去重/幂等验证
- [x] 4.3 在 `docs/mainline-contract-test-index.md` 登记新增合同测试条目

## 5. Documentation Sync

- [x] 5.1 更新 `README.md`：说明 skill trigger scoring 默认策略、tie-break、低置信度抑制与非目标（embedding TODO）
- [x] 5.2 更新 `docs/runtime-config-diagnostics.md`：新增 `skill.trigger_scoring` 配置索引与校验语义
- [x] 5.3 更新 `docs/v1-acceptance.md`：新增 D1 验收项（评分契约与配置生效）
- [x] 5.4 更新 `docs/development-roadmap.md`：标记该提案落点与 embedding 后续 TODO

## 6. Validation

- [x] 6.1 执行 `go test ./...` 并修复回归
- [x] 6.2 执行 `go test -race ./...` 并确认并发安全基线
- [x] 6.3 执行 `golangci-lint run --config .golangci.yml` 并修复问题
