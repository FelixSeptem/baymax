# Modular E2E Review Matrix

更新时间：2026-03-13

## 范围

本矩阵用于治理型迭代（模块职责校验 + 主干串联语义校验）。

## 模块评审清单

| 模块 | 职责边界检查 | 错误语义检查 | 并发安全检查 | 可观测一致性检查 |
| --- | --- | --- | --- | --- |
| `core/*` | 状态机只编排，不下沉 provider 细节 | `types.ErrorClass` 与终止语义一致 | 取消传播、迭代边界无竞态 | run/timeline 序列一致 |
| `context/*` | 仅策略编排，不直接依赖 provider SDK | stage fail-fast/best-effort 行为一致 | CA3 状态更新与 spill/swap 单进程一致 | assemble 诊断字段完整 |
| `model/*` | provider SDK 细节集中在 model 层 | provider_reason 与 ErrorClass 映射一致 | streaming 事件顺序稳定 | model 事件可关联 run/phase |
| `runtime/*` | 全局配置/诊断不反向依赖 transport | reload/record 错误可解释 | single-writer + 幂等去重 | diagnostics 与 timeline 聚合一致 |
| `observability/*` | 事件到诊断映射单向稳定 | 错误类与状态枚举不漂移 | 并发写入无重复计数 | trace/run/sequence 关联完整 |

## 主干流程串联清单

| 流程 | 期望语义 | 对应契约测试 |
| --- | --- | --- |
| `Run` | 成功/失败终止语义稳定；run.finished 必发 | `core/runner/runner_test.go::TestRunNormalCompletionAndEvents`, `TestRunTimeoutAbort` |
| `Stream` | delta 聚合稳定；fail-fast 不吞错 | `core/runner/runner_test.go::TestStreamForwardsDelta`, `TestStreamFailFastWithErrModel` |
| `tool-loop` | tool feedback 回写、continue/fail-fast 策略一致 | `core/runner/runner_test.go::TestRunToolLoopSuccess`, `TestRunToolFailurePolicy` |
| `CA2 stage2` | rules 路由与 stage policy 行为可预测 | `core/runner/runner_test.go::TestRunCA2BestEffortKeepsModelPath`, `TestStreamCA2FailFastStopsBeforeModel` |
| `CA3 pressure/recovery` | zone 与动作语义 Run/Stream 一致 | `core/runner/runner_test.go::TestRunAndStreamCA3PressureSemanticsEquivalent`, `context/assembler/assembler_test.go::TestAssemblerCA3SpillIdempotentAcrossRetry` |

## Findings（本次收敛）

### P0

1. 仓库存在临时备份产物 `context/assembler/assembler.go.398874324043069051` 与 `context/assembler/ca3.go.6601493124856893574`。
- 风险：误导检索、评审噪音、潜在错误回流。
- 修复：删除备份文件；新增仓库卫生检查脚本并接入质量门禁。

### P1

1. CA3 `TokenCounter` 在 `Run/Stream` 的 selected-provider 注入路径缺少契约测试。
- 风险：fallback 场景可能错误使用非选中 provider 计数逻辑。
- 修复：新增 runner 契约测试，验证 fallback 后 token counter 来源与模型字段传递。

2. provider token-count 归一化缺少回归测试（Gemini role 映射、OpenAI unsupported 语义）。
- 风险：后续改动可能引入计数语义漂移。
- 修复：新增 provider 级测试覆盖关键路径。

### P2

1. 缺少“主干流程 -> 测试”显式索引，难以快速核验覆盖完整性。
- 修复：新增测试索引文档并在 README 建立导航。

2. 质量门禁文档未显式包含仓库卫生检查。
- 修复：同步 README/docs 与脚本语义。

## 闭环状态

- P0：已修复
- P1：已修复
- P2：已修复
