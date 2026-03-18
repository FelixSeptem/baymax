# Mainline Contract Test Index

更新时间：2026-03-18

## 目标

提供主干流程与契约测试用例的一一映射，作为质量门禁核对入口。

## 索引

| 主干流程 | 正向场景 | 异常/降级场景 |
| --- | --- | --- |
| Run | `core/runner/runner_test.go::TestRunNormalCompletionAndEvents` | `core/runner/runner_test.go::TestRunTimeoutAbort` |
| Stream | `core/runner/runner_test.go::TestStreamForwardsDelta` | `core/runner/runner_test.go::TestStreamFailFastWithErrModel` |
| Tool Loop | `core/runner/runner_test.go::TestRunToolLoopSuccess` | `core/runner/runner_test.go::TestRunToolFailurePolicy` |
| CA2 Stage2 | `core/runner/runner_test.go::TestRunCA2BestEffortKeepsModelPath` | `core/runner/runner_test.go::TestStreamCA2FailFastStopsBeforeModel` |
| CA2 Agentic Routing | `core/runner/runner_test.go::TestCA2AgenticRoutingRunAndStreamSemanticEquivalent` | `core/runner/runner_test.go::TestCA2AgenticRoutingFallbackRunAndStreamSemanticEquivalent` |
| CA3 Pressure | `core/runner/runner_test.go::TestRunAndStreamCA3PressureSemanticsEquivalent` | `context/assembler/assembler_test.go::TestAssemblerCA3EmergencyRejectsLowPriorityStage2` |
| CA3 Compaction Semantic | `context/assembler/assembler_test.go::TestAssemblerCA3SemanticCompactionUsesModelClient` | `context/assembler/assembler_test.go::TestAssemblerCA3SemanticCompactionQualityGateBestEffortFallback` |
| Action Gate H2 | `core/runner/runner_test.go::TestActionGateAllowPathKeepsToolExecution` | `core/runner/runner_test.go::TestActionGateRunAndStreamTimeoutSemanticsEquivalent` |
| Action Gate H4 Parameter Rules | `core/runner/runner_test.go::TestActionGateParameterRulePriorityOverKeyword` | `core/runner/runner_test.go::TestActionGateParameterRuleRunAndStreamTimeoutSemanticsEquivalent` |
| Security Policy S2 | `core/runner/runner_test.go::TestSecurityPolicyContractPermissionAllowAndDeny` | `core/runner/runner_test.go::TestSecurityPolicyContractRateLimitDeny` |
| Security Event S3 | `core/runner/runner_test.go::TestSecurityEventContractPermissionDenyTriggersCallback` | `core/runner/runner_test.go::TestSecurityEventContractCallbackFailureDoesNotChangeDenyOutcome` |
| Security Delivery S4 | `core/runner/runner_test.go::TestSecurityDeliveryContractRetryBudget` | `core/runner/runner_test.go::TestSecurityDeliveryContractCircuitTransitions` |
| Runner Concurrency Baseline R5 | `core/runner/runner_test.go::TestRunBackpressureBlockDiagnosticsAndTimeline` | `core/runner/runner_test.go::TestRunAndStreamCancelPropagationSemanticsEquivalent` |
| Backpressure Drop-Low-Priority R6 | `tool/local/registry_test.go::TestDispatcherDropLowPriorityDropsConfiguredLowCalls` | `core/runner/runner_test.go::TestRunBackpressureDropLowPriorityAllDroppedFailsFast` |
| Backpressure Drop-Low-Priority R7 | `tool/local/registry_test.go::TestDispatcherDropLowPriorityMarksMCPAndSkillPhase` | `core/runner/runner_test.go::TestRunBackpressureDropLowPriorityMCPAndSkillAllDroppedFailsFast` |
| Action Timeline Cross-Run Trend H16 | `runtime/diagnostics/store_test.go::TestStoreTimelineTrendsLastNRunsAndTimeWindow` | `runtime/diagnostics/store_test.go::TestStoreTimelineTrendsIdempotentReplayAndEmptyWindow` |
| CA2 External Observability E2 | `runtime/diagnostics/store_test.go::TestStoreCA2ExternalTrendsRunStreamSemanticEquivalent` | `runtime/diagnostics/store_test.go::TestStoreCA2ExternalTrendsThresholdSignalsAndErrorLayerExtension` |
| Provider Fallback + CA3 Token Counter | `core/runner/runner_test.go::TestRunProviderFallbackUsesSelectedTokenCounterForCA3` | `core/runner/runner_test.go::TestStreamProviderFallbackUsesSelectedTokenCounterForCA3` |
| Provider Token Count Normalization | `model/gemini/client_test.go::TestBuildTokenContentsNormalizesRolesAndKeepsInput` | `model/openai/client_test.go::TestCountTokensReturnsUnsupportedError` |
| Skill Trigger Scoring D1 | `skill/loader/loader_test.go::TestCompileSemanticTieBreakUsesHighestPriority` | `skill/loader/loader_test.go::TestCompileDefaultSuppressesLowConfidenceSemanticMatch` |
| Skill Trigger Scoring D2 | `skill/loader/loader_test.go::TestCompileLexicalPlusEmbeddingWeightedScore` | `skill/loader/loader_test.go::TestCompileLexicalPlusEmbeddingFallbackReasons` |
| Skill Trigger Scoring D3 | `skill/loader/loader_test.go::TestCompileMixedCJKENLexicalTokenization` | `skill/loader/loader_test.go::TestCompileTopKBudgetAndExplicitBypass` |
| Skill Trigger Run/Stream Equivalence D3 | `skill/loader/loader_test.go::TestCompileMultilingualBudgetRunAndStreamSemanticEquivalent` | `runtime/config/manager_test.go::TestManagerSkillTriggerLexicalBudgetInvalidReloadRollsBack` |
| A2A Delivery/Version Negotiation A4 | `a2a/interop_test.go::TestSSEReconnectAndSubscribeReasons` | `a2a/interop_test.go::TestRunAndStreamSemanticEquivalenceForFallbackAndVersionMismatch` |
| Composed Orchestration A5 | `integration/composed_orchestration_contract_test.go::TestWorkflowA2ARemoteStepRunStreamContract` | `integration/composed_orchestration_contract_test.go::TestComposedA2AAndMCPBoundaryRegression` |
| Teams Mixed Local+Remote A5 | `orchestration/teams/engine_test.go::TestParallelMixedLocalAndRemoteExecution` | `orchestration/teams/engine_test.go::TestMixedCancellationConvergence` |
| Workflow A2A Step A5 | `orchestration/workflow/engine_test.go::TestA2ARetryAndTimeoutSemantics` | `orchestration/workflow/engine_test.go::TestA2ACheckpointResumeSemantics` |
| R4 Shared Contract Freeze Gate | `tool/contributioncheck/multi_agent_contract_test.go::TestMultiAgentSharedContractSnapshotPass` | `tool/contributioncheck/multi_agent_contract_test.go::TestValidateMultiAgentSharedContractDetectsViolations` |
| CA3 Semantic Embedding Adapter E3 | `context/assembler/assembler_test.go::TestAssemblerCA3SemanticCompactionHybridScoreUsesCosineWeight` | `context/assembler/assembler_test.go::TestAssemblerCA3SemanticCompactionEmbeddingFailureFailFast` |
| CA3 Reranker And Tuning E4 | `context/assembler/assembler_test.go::TestAssemblerCA3RerankerBestEffortFallback` | `context/assembler/assembler_test.go::TestAssemblerCA3RerankerFailFast` |
| CA3 Threshold Governance E5 | `context/assembler/assembler_test.go::TestAssemblerCA3RerankerGovernanceEnforceVsDryRun` | `context/assembler/assembler_test.go::TestAssemblerCA3RerankerGovernanceModeFailurePolicy` |
| CA3 Governance Run/Stream Equivalence E5 | `core/runner/runner_test.go::TestRunAndStreamCA3GovernanceSemanticsEquivalent` | `context/assembler/assembler_test.go::TestAssemblerCA3RerankerGovernanceRolloutMatchDeterministic` |
| CA3 Threshold Tuning Toolkit E4 | `context/assembler/threshold_tuning_test.go::TestRunThresholdTuningProducesDeterministicRecommendation` | `context/assembler/threshold_tuning_test.go::TestRunThresholdTuningRejectsUnsupportedSchema` |

## 使用方式

1. 变更完成前，确保以上用例在 `go test ./...` 中通过。
2. 合并前，执行 `go test -race ./...` 验证并发安全基线。
3. 质量门禁脚本执行时应同时包含仓库卫生检查、lint 与安全扫描。
4. 基准回归可执行：`integration/benchmark_test.go::BenchmarkCA2ExternalRetrieverTrendAggregation`、`integration/benchmark_test.go::BenchmarkCA3SemanticCompactionLatency`、`integration/benchmark_test.go::BenchmarkCA3SemanticCompactionLatencyEmbeddingEnabled`、`integration/benchmark_test.go::BenchmarkCA3SemanticCompactionLatencyRerankerGovernanceEnabled`。
