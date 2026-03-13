## 1. CA3 Pressure Control Core

- [x] 1.1 在 `context/assembler` 实现分级压力响应状态机（safe/comfort/warning/danger/emergency）并保持 Goldilocks 目标区间语义
- [x] 1.2 实现双触发阈值判定（百分比 + 绝对 token），并支持阶段级阈值配置
- [x] 1.3 落地 batch squash/prune 最小规则策略（关键词/访问频率/最近使用）
- [x] 1.4 支持 `critical`/`immutable` 标记，确保 squash/prune 不破坏受保护内容

## 2. Spill/Swap And Recovery

- [x] 2.1 实现文件型 spill/swap 后端（含 `origin_ref` 持久化与回填）
- [x] 2.2 在紧急区启用保护模式：默认拒绝低优先级新加载，高优先级降级通过
- [x] 2.3 实现单进程 cancel/retry/replay 一致性恢复流程
- [x] 2.4 预留 DB/对象存储接口（仅接口定义，不实现后端）

## 3. Runtime Config And Diagnostics

- [x] 3.1 扩展 `runtime/config` 支持 CA3 配置字段并保持 `env > file > default`
- [x] 3.2 扩展 `runtime/diagnostics` run 记录字段：分区停留时长、触发次数、压缩率、溢出次数、回填次数
- [x] 3.3 在 `observability/event` 接入 CA3 字段映射，保持旧字段兼容

## 4. Tests And Quality Gates

- [x] 4.1 新增/更新契约测试：Run/Stream 压力分级决策结果语义一致
- [x] 4.2 新增恢复与幂等测试：cancel/retry/replay 不出现状态撕裂或重复副作用
- [x] 4.3 新增配置校验测试：阈值越界/冲突时 fail-fast
- [x] 4.4 执行质量门禁：`go test ./...`、`go test -race ./...`、`golangci-lint run --config .golangci.yml`

## 5. Documentation Alignment

- [x] 5.1 更新 `README.md`（CA3 能力、限制、配置摘要）
- [x] 5.2 更新 `docs/context-assembler-phased-plan.md`（与实现和默认阈值保持一致）
- [x] 5.3 更新 `docs/runtime-config-diagnostics.md`（CA3 配置与诊断字段）
- [x] 5.4 更新 `docs/development-roadmap.md`（标记进展与后续 TODO）
- [x] 5.5 执行 docs 一致性检查（`scripts/check-docs-consistency.ps1`）
