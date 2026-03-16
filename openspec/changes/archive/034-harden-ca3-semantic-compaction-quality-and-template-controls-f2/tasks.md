## 1. CA3 Quality Gate

- [x] 1.1 为 semantic compaction 增加规则评分（coverage/compression/validity）
- [x] 1.2 增加 `quality.threshold` 判定与失败分支
- [x] 1.3 对质量失败统一映射 `fallback_reason`（如 `quality_below_threshold`）

## 2. Template Controls

- [x] 2.1 增加 `semantic_template.prompt` 与 `allowed_placeholders` 配置
- [x] 2.2 增加模板校验（非空、占位符平衡、白名单限制）
- [x] 2.3 启动与热更新保持 fail-fast 校验语义

## 3. Embedding Hook (Placeholder)

- [x] 3.1 增加 `embedding.enabled` 与 `embedding.selector` 配置项
- [x] 3.2 保持 hook-only：未绑定 adapter 时不改变主路径行为
- [x] 3.3 在质量 reason 中增加 hook 未绑定可观测标记

## 4. Diagnostics and Contracts

- [x] 4.1 增加 `ca3_compaction_fallback_reason`
- [x] 4.2 增加 `ca3_compaction_quality_score`
- [x] 4.3 增加 `ca3_compaction_quality_reason`
- [x] 4.4 保证 Run/Stream 对等输入下 compaction 语义一致

## 5. Benchmarks and Docs

- [x] 5.1 增补 CA3 semantic benchmark 基线
- [x] 5.2 更新 `README.md` 与 `docs/runtime-config-diagnostics.md`
- [x] 5.3 更新 `docs/development-roadmap.md` 与 `docs/v1-acceptance.md`

> 注：本 tasks 为归档异常后恢复重建版本（Recovered），任务完成状态依据仓库现状与文档完成标记回填。
