# Mainline Contract Test Index

更新时间：2026-03-19

## 目标

提供主干流程与契约测试用例的一一映射，作为质量门禁核对入口。

A12/A13 收口兼容语义参考：`docs/v1-acceptance.md` 中 compatibility window（`additive + nullable + default`）条款。

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
| Composed Remote Failure A5 | `orchestration/teams/engine_test.go::TestRemoteFailureAggregatesAndTimelineReasons` | `orchestration/workflow/engine_test.go::TestA2AFailureStillUsesDispatchReasonNamespace` |
| Composed Config/Reload A5 | `runtime/config/config_test.go::TestTeamsConfigEnvOverridePrecedence` | `runtime/config/manager_test.go::TestManagerWorkflowRemoteInvalidReloadRollsBack` |
| Composed Diagnostics Replay A5 | `runtime/diagnostics/store_test.go::TestStoreRunTeamsAggregateReplayIsIdempotent` | `runtime/diagnostics/store_test.go::TestStoreRunWorkflowAggregateReplayIsIdempotent` |
| Scheduler Crash Takeover A6 | `integration/distributed_subagent_scheduler_contract_test.go::TestWorkerCrashLeaseExpiryTakeover` | `integration/distributed_subagent_scheduler_contract_test.go::TestA2ASchedulerRetryAndErrorLayerNormalization` |
| Scheduler Idempotent Replay A6 | `integration/distributed_subagent_scheduler_contract_test.go::TestSchedulerDuplicateSubmitResultReplayIdempotency` | `runtime/diagnostics/store_test.go::TestStoreRunSchedulerSubagentAggregateReplayIsIdempotent` |
| Scheduler Run/Stream Equivalence A6 | `integration/distributed_subagent_scheduler_contract_test.go::TestSchedulerManagedRunStreamSemanticEquivalence` | `integration/distributed_subagent_scheduler_contract_test.go::TestSchedulerManagedRunStreamSemanticEquivalence` |
| A2A Scheduler Retry A6 | `orchestration/scheduler/a2a_adapter_test.go::TestExecuteClaimWithA2ASuccess` | `orchestration/scheduler/a2a_adapter_test.go::TestExecuteClaimWithA2ASubmitTransportErrorIsRetryable` |
| Scheduler Recovery Takeover A7 | `integration/scheduler_recovery_contract_test.go::TestSchedulerRecoveryCrashLeaseExpiryTakeover` | `integration/scheduler_recovery_contract_test.go::TestSchedulerRecoveryTimelineCorrelationRequiredFields` |
| Scheduler Recovery Idempotency A7 | `integration/scheduler_recovery_contract_test.go::TestSchedulerRecoveryDuplicateSubmitCommitIdempotency` | `runtime/diagnostics/store_test.go::TestStoreRunSchedulerSubagentAggregateReplayIsIdempotent` |
| Scheduler Recovery Run/Stream A7 | `integration/scheduler_recovery_contract_test.go::TestSchedulerRecoveryRunStreamSemanticEquivalence` | `integration/scheduler_recovery_contract_test.go::TestSchedulerRecoveryRunStreamSemanticEquivalence` |
| Composer Run/Stream Equivalence A8 | `integration/composer_contract_test.go::TestComposerContractRunStreamSemanticEquivalence` | `orchestration/composer/composer_test.go::TestComposerSchedulerReloadAppliesOnNextAttemptOnly` |
| Composer Scheduler Fallback A8 | `integration/composer_contract_test.go::TestComposerContractSchedulerFallbackToMemory` | `integration/composer_contract_test.go::TestComposerContractSchedulerFallbackToMemory` |
| Composer Child Replay Idempotency A8 | `integration/composer_contract_test.go::TestComposerContractTakeoverReplayIdempotency` | `orchestration/composer/composer_test.go::TestComposerGuardrailFailFastEmitsBudgetRejectAndSummary` |
| Composer Recovery Cross-Session A9 | `integration/composer_recovery_contract_test.go::TestComposerRecoveryCrossSessionResumeSuccess` | `orchestration/composer/recovery_store_test.go::TestFileRecoveryStoreRoundTripAndDuplicateLoad` |
| Composer Recovery Replay Idempotency A9 | `integration/composer_recovery_contract_test.go::TestComposerRecoveryReplayIdempotent` | `runtime/diagnostics/store_test.go::TestStoreRunRecoveryAggregateReplayIsIdempotent` |
| Composer Recovery Conflict Fail-Fast A9 | `integration/composer_recovery_contract_test.go::TestComposerRecoveryConflictFailFast` | `orchestration/composer/recovery_store_test.go::TestFileRecoveryStoreCorruptSnapshotFailsFast` |
| Scheduler QoS Fairness A10 | `integration/scheduler_qos_contract_test.go::TestSchedulerQoSPriorityFairnessAndAntiStarvation` | `integration/scheduler_qos_contract_test.go::TestSchedulerQoSPriorityFairnessAndAntiStarvation` |
| Scheduler QoS Backoff And DLQ A10 | `integration/scheduler_qos_contract_test.go::TestSchedulerQoSRetryBackoffDeadLetterAndReplayIdempotency` | `integration/scheduler_qos_contract_test.go::TestSchedulerQoSRetryBackoffDeadLetterAndReplayIdempotency` |
| Scheduler QoS Run/Stream Equivalence A10 | `integration/scheduler_qos_contract_test.go::TestSchedulerQoSRunStreamSemanticEquivalence` | `integration/scheduler_qos_contract_test.go::TestSchedulerQoSRunStreamSemanticEquivalence` |
| Sync Invocation Coverage A11 | `integration/sync_invocation_contract_test.go::TestSyncInvocationContractWorkflowTeamsSchedulerComposerConsistency` | `integration/sync_invocation_contract_test.go::TestSyncInvocationContractWorkflowTeamsSchedulerComposerConsistency` |
| Sync Invocation Run/Stream Equivalence A11 | `integration/sync_invocation_contract_test.go::TestSyncInvocationContractRunStreamRemoteAggregateEquivalence` | `integration/sync_invocation_contract_test.go::TestSyncInvocationContractRunStreamRemoteAggregateEquivalence` |
| Sync Invocation Scheduler Canceled Mapping A11 | `integration/sync_invocation_contract_test.go::TestSyncInvocationContractSchedulerCanceledTerminalMappingAndRetryable` | `orchestration/scheduler/a2a_adapter_test.go::TestExecuteClaimWithA2ACanceledTerminalMapsToFailedDeterministically` |
| Async Reporting Delivery A12 | `integration/async_reporting_contract_test.go::TestAsyncReportingContractDeliveryMatrix` | `integration/async_reporting_contract_test.go::TestAsyncReportingContractDeliveryMatrix` |
| Async Reporting Dedup Replay A12 | `integration/async_reporting_contract_test.go::TestAsyncReportingContractDedupAndReplayIdempotency` | `integration/async_reporting_contract_test.go::TestAsyncReportingContractDedupAndReplayIdempotency` |
| Async Reporting Run/Stream Equivalence A12 | `integration/async_reporting_contract_test.go::TestAsyncReportingContractRunStreamEquivalence` | `integration/async_reporting_contract_test.go::TestAsyncReportingContractRunStreamEquivalence` |
| Async Reporting Recovery Replay A12 | `integration/async_reporting_contract_test.go::TestAsyncReportingContractRecoveryReplayNoInflation` | `integration/async_reporting_contract_test.go::TestAsyncReportingContractRecoveryReplayNoInflation` |
| Delayed Dispatch Eligibility A13 | `integration/delayed_dispatch_contract_test.go::TestDelayedDispatchContractEarlyClaimBlockedThenReady` | `orchestration/scheduler/store_test.go::TestSchedulerClaimComposesDelayedAndRetryGate` |
| Delayed Dispatch Recovery A13 | `integration/delayed_dispatch_contract_test.go::TestDelayedDispatchContractRecoveryNoEarlyClaim` | `integration/delayed_dispatch_contract_test.go::TestDelayedDispatchContractRecoveryNoEarlyClaim` |
| Delayed Dispatch Run/Stream Equivalence A13 | `integration/delayed_dispatch_contract_test.go::TestDelayedDispatchContractRunStreamSemanticEquivalence` | `integration/composer_contract_test.go::TestComposerContractDelayedChildRunStreamEquivalence` |
| Delayed Dispatch Async Compatibility A13 | `integration/delayed_dispatch_contract_test.go::TestDelayedDispatchContractAsyncReportingCompatibility` | `integration/delayed_dispatch_contract_test.go::TestDelayedDispatchContractAsyncReportingCompatibility` |
| Tail Governance Cross-Mode Matrix A14 | `integration/tail_governance_contract_test.go::TestTailGovernanceA14CrossModeMatrixRunStream` | `integration/tail_governance_contract_test.go::TestTailGovernanceA14CrossModeMatrixRunStream` |
| Tail Governance QoS+Recovery A14 | `integration/tail_governance_contract_test.go::TestTailGovernanceA14QoSRecoveryRunStreamSemanticEquivalence` | `integration/tail_governance_contract_test.go::TestTailGovernanceA14QoSRecoveryRunStreamSemanticEquivalence` |
| Tail Governance Parser Compatibility A14 | `observability/event/runtime_recorder_test.go::TestRuntimeRecorderA14ParserCompatibilityAdditiveNullableDefault` | `runtime/diagnostics/store_test.go::TestStoreRunAsyncDelayedAggregateReplayIsIdempotent` |
| Tail Governance Async+Delayed Replay A14 | `integration/delayed_dispatch_contract_test.go::TestDelayedDispatchContractAsyncDelayedReplayNoInflation` | `runtime/diagnostics/store_test.go::TestStoreRunAsyncDelayedAggregateReplayIsIdempotent` |
| Workflow Graph Composability Determinism A15 | `integration/workflow_graph_composability_contract_test.go::TestWorkflowGraphComposabilityA15ExpansionDeterminismAndCanonicalIDs` | `orchestration/workflow/composability_test.go::TestComposablePlanAndRunUseCanonicalExpandedIDs` |
| Workflow Graph Composability Compile Fail-Fast A15 | `integration/workflow_graph_composability_contract_test.go::TestWorkflowGraphComposabilityA15CompileFailFastMatrix` | `orchestration/workflow/composability_test.go::TestComposableCompileFailuresArePreDispatchForRunAndStream` |
| Workflow Graph Composability Run/Stream+Resume A15 | `integration/workflow_graph_composability_contract_test.go::TestWorkflowGraphComposabilityA15RunStreamEquivalenceAndResumeConsistency` | `orchestration/workflow/composability_test.go::TestComposableResumeKeepsExpandedStepDeterminism` |
| Workflow Graph Composability Composer Path A15 | `integration/workflow_graph_composability_contract_test.go::TestWorkflowGraphComposabilityA15ComposerManagedRemoteStep` | `integration/workflow_graph_composability_contract_test.go::TestWorkflowGraphComposabilityA15ComposerManagedRemoteStep` |
| Collaboration Primitives Sync/Async/Delayed A16 | `integration/collaboration_primitives_contract_test.go::TestCollaborationPrimitivesA16SyncMode` | `integration/collaboration_primitives_contract_test.go::TestCollaborationPrimitivesA16AsyncReportingMode` |
| Collaboration Primitives Delayed + Timeline A16 | `integration/collaboration_primitives_contract_test.go::TestCollaborationPrimitivesA16DelayedDispatchMode` | `integration/collaboration_primitives_contract_test.go::TestCollaborationPrimitivesA16TimelineReasonAndCorrelation` |
| Collaboration Primitives Run/Stream + Recovery A16 | `integration/collaboration_primitives_contract_test.go::TestCollaborationPrimitivesA16RunStreamEquivalence` | `integration/collaboration_primitives_contract_test.go::TestCollaborationPrimitivesA16ReplayRecoveryConsistency` |
| Recovery Boundary Crash/Restart/Replay/Timeout Matrix A17 | `integration/recovery_boundary_contract_test.go::TestRecoveryBoundaryA17CrashRestartReplayTimeoutMatrix` | `integration/composer_recovery_contract_test.go::TestComposerRecoveryBoundaryViolationClassifiedAsConflict` |
| Recovery Boundary Run/Stream Equivalence A17 | `integration/recovery_boundary_contract_test.go::TestRecoveryBoundaryA17RunStreamEquivalence` | `integration/recovery_boundary_contract_test.go::TestRecoveryBoundaryA17RunStreamEquivalence` |
| Recovery Boundary Replay Idempotency A17 | `integration/recovery_boundary_contract_test.go::TestRecoveryBoundaryA17ReplayIdempotency` | `integration/composer_recovery_contract_test.go::TestComposerRecoveryReplayIdempotent` |
| Unified Query A18 Filters + Empty Task Semantics | `runtime/diagnostics/store_test.go::TestStoreUnifiedRunQueryAndSemantics` | `integration/unified_query_contract_test.go::TestUnifiedQueryContractUnmatchedTaskIDEmptySet` |
| Unified Query A18 Pagination + Cursor | `runtime/diagnostics/store_test.go::TestStoreUnifiedRunQueryValidationAndPagingBounds` | `runtime/diagnostics/store_test.go::TestStoreUnifiedRunQueryCursorDeterministicAndFailFast` |
| Unified Query A18 Replay Idempotent Summary Query | `integration/unified_query_contract_test.go::TestUnifiedQueryContractReplayIdempotentSummaries` | `integration/unified_query_contract_test.go::TestUnifiedQueryContractReplayIdempotentSummaries` |
| Multi-Agent Mainline Performance A19 Benchmark Matrix | `integration/benchmark_test.go::BenchmarkMultiAgentMainlineSyncInvocation`、`integration/benchmark_test.go::BenchmarkMultiAgentMainlineAsyncReporting`、`integration/benchmark_test.go::BenchmarkMultiAgentMainlineDelayedDispatch`、`integration/benchmark_test.go::BenchmarkMultiAgentMainlineRecoveryReplay` | `scripts/check-multi-agent-performance-regression.sh`、`scripts/check-multi-agent-performance-regression.ps1` |
| Multi-Agent Mainline Performance A19 Gate Integration | `scripts/check-quality-gate.sh` | `scripts/check-quality-gate.ps1` |
| Full-Chain Reference Example A20 Smoke Gate | `scripts/check-full-chain-example-smoke.sh` | `scripts/check-full-chain-example-smoke.ps1` |
| Full-Chain Reference Example A20 Quality Path | `scripts/check-quality-gate.sh` | `scripts/check-quality-gate.ps1` |
| External Adapter Template A21 Docs Consistency | `scripts/check-docs-consistency.sh` | `scripts/check-docs-consistency.ps1` |
| External Adapter Template A21 Contribution Traceability | `tool/contributioncheck/adapter_docs_test.go::TestAdapterOnboardingDocsConsistency` | `tool/contributioncheck/adapter_docs_test.go::TestAdapterOnboardingDocsConsistency` |
| External Adapter Conformance A22 Matrix | `integration/adapterconformance/harness_test.go::TestAdapterConformanceMCPNormalizationAndFailFast`、`integration/adapterconformance/harness_test.go::TestAdapterConformanceModelRunStreamAndDowngrade`、`integration/adapterconformance/harness_test.go::TestAdapterConformanceToolInvocationAndFailFast` | `integration/adapterconformance/harness_test.go::TestAdapterConformanceTemplateTraceabilityAndDriftGuard` |
| External Adapter Conformance A22 Gate Path | `scripts/check-adapter-conformance.sh` | `scripts/check-adapter-conformance.ps1` |
| Adapter Scaffold A23 Determinism + Conflict | `adapter/scaffold/scaffold_test.go::TestBuildPlanDeterministic`、`adapter/scaffold/scaffold_test.go::TestGenerateConflictFailFastNoPartialWrite`、`adapter/scaffold/scaffold_test.go::TestGenerateForceOverwrite` | `adapter/scaffold/scaffold_test.go::TestDefaultOutputPathWhenOutputOmitted` |
| Adapter Scaffold A23 Bootstrap Mapping + Offline Executable | `adapter/scaffold/scaffold_test.go::TestBuildPlanIncludesCategoryBootstrapHints` | `adapter/scaffold/scaffold_test.go::TestGeneratedConformanceBootstrapOfflineExecutable` |
| Adapter Scaffold A23 Drift Gate Path | `scripts/check-adapter-scaffold-drift.sh` | `scripts/check-adapter-scaffold-drift.ps1` |
| Adapter Manifest Contract A26 Core Validation | `adapter/manifest/manifest_test.go::TestParseAndValidateManifestSuccess` | `adapter/manifest/manifest_test.go::TestParseManifestDetectsMissingFieldDeterministically` |
| Adapter Manifest Runtime Activation A26 | `integration/adapterconformance/harness_test.go::TestAdapterConformanceManifestActivationSuccess` | `integration/adapterconformance/harness_test.go::TestAdapterConformanceManifestRequiredCapabilityFailFast` |
| Adapter Manifest Scaffold/Conformance Alignment A26 | `integration/adapterconformance/harness_test.go::TestAdapterConformanceManifestProfileAlignmentForFixtures` | `adapter/scaffold/scaffold_test.go::TestBuildPlanIncludesCategoryBootstrapHints` |
| Adapter Manifest Contract A26 Gate Path | `scripts/check-adapter-manifest-contract.sh` | `scripts/check-adapter-manifest-contract.ps1` |
| Pre-1 Governance A24 Docs Consistency | `tool/contributioncheck/governance_docs_test.go::TestPre1GovernanceDocsConsistency` | `tool/contributioncheck/governance_docs_test.go::TestValidatePre1GovernanceDocsDetectsStageConflict` |
| Pre-1 Governance A24 Gate Path | `scripts/check-docs-consistency.sh` | `scripts/check-docs-consistency.ps1` |
| Pre-1 Governance A24 Quality Path | `scripts/check-quality-gate.sh` | `scripts/check-quality-gate.ps1` |
| Status Parity Governance A25 | `tool/contributioncheck/status_parity_test.go::TestReleaseStatusParityDocsConsistency` | `tool/contributioncheck/status_parity_test.go::TestValidateStatusParityDetectsConflict` |
| Core Module README Richness A25 | `tool/contributioncheck/module_readme_richness_test.go::TestCoreModuleReadmeRichnessBaseline` | `tool/contributioncheck/module_readme_richness_test.go::TestValidateCoreModuleReadmeRichnessDetectsMissingSection` |
| Status Parity + README Richness A25 Gate Path | `scripts/check-docs-consistency.sh` | `scripts/check-docs-consistency.ps1` |
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
4. 基准回归可执行：`integration/benchmark_test.go::BenchmarkCA2ExternalRetrieverTrendAggregation`、`integration/benchmark_test.go::BenchmarkCA3SemanticCompactionLatency`、`integration/benchmark_test.go::BenchmarkCA3SemanticCompactionLatencyEmbeddingEnabled`、`integration/benchmark_test.go::BenchmarkCA3SemanticCompactionLatencyRerankerGovernanceEnabled`、`integration/benchmark_test.go::BenchmarkMultiAgentMainlineSyncInvocation`、`integration/benchmark_test.go::BenchmarkMultiAgentMainlineAsyncReporting`、`integration/benchmark_test.go::BenchmarkMultiAgentMainlineDelayedDispatch`、`integration/benchmark_test.go::BenchmarkMultiAgentMainlineRecoveryReplay`。
