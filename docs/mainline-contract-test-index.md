# Mainline Contract Test Index

更新时间：2026-03-16

## 目标

提供主干流程与契约测试用例的一一映射，作为质量门禁核对入口。

## 索引

| 主干流程 | 正向场景 | 异常/降级场景 |
| --- | --- | --- |
| Run | `core/runner/runner_test.go::TestRunNormalCompletionAndEvents` | `core/runner/runner_test.go::TestRunTimeoutAbort` |
| Stream | `core/runner/runner_test.go::TestStreamForwardsDelta` | `core/runner/runner_test.go::TestStreamFailFastWithErrModel` |
| Tool Loop | `core/runner/runner_test.go::TestRunToolLoopSuccess` | `core/runner/runner_test.go::TestRunToolFailurePolicy` |
| CA2 Stage2 | `core/runner/runner_test.go::TestRunCA2BestEffortKeepsModelPath` | `core/runner/runner_test.go::TestStreamCA2FailFastStopsBeforeModel` |
| CA3 Pressure | `core/runner/runner_test.go::TestRunAndStreamCA3PressureSemanticsEquivalent` | `context/assembler/assembler_test.go::TestAssemblerCA3EmergencyRejectsLowPriorityStage2` |
| Action Gate H2 | `core/runner/runner_test.go::TestActionGateAllowPathKeepsToolExecution` | `core/runner/runner_test.go::TestActionGateRunAndStreamTimeoutSemanticsEquivalent` |
| Action Gate H4 Parameter Rules | `core/runner/runner_test.go::TestActionGateParameterRulePriorityOverKeyword` | `core/runner/runner_test.go::TestActionGateParameterRuleRunAndStreamTimeoutSemanticsEquivalent` |
| Provider Fallback + CA3 Token Counter | `core/runner/runner_test.go::TestRunProviderFallbackUsesSelectedTokenCounterForCA3` | `core/runner/runner_test.go::TestStreamProviderFallbackUsesSelectedTokenCounterForCA3` |
| Provider Token Count Normalization | `model/gemini/client_test.go::TestBuildTokenContentsNormalizesRolesAndKeepsInput` | `model/openai/client_test.go::TestCountTokensReturnsUnsupportedError` |

## 使用方式

1. 变更完成前，确保以上用例在 `go test ./...` 中通过。
2. 合并前，执行 `go test -race ./...` 验证并发安全基线。
3. 质量门禁脚本执行时应同时包含仓库卫生检查、lint 与安全扫描。
