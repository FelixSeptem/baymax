# Runtime Config & Diagnostics Naming Migration Inventory (A63 10.1)

## Scope

This inventory tracks CA-era keys/fields in:

- `runtime/config`
- `runtime/diagnostics` ingest path (via `observability/event.RuntimeRecorder`)
- `core/runner` `run.finished` payload emission

## Config Key Mapping (Context Assembler)

- Semantic primary: `context_assembler.stage2_routing_and_disclosure`
- Legacy alias: `context_assembler.ca2`
- Compatibility behavior: semantic and legacy keys are both accepted; when both are present with conflicting values, semantic primary takes precedence.

- Semantic primary: `context_assembler.pressure_compaction_and_swapback`
- Legacy alias: `context_assembler.ca3`
- Compatibility behavior: semantic and legacy keys are both accepted; when both are present with conflicting values, semantic primary takes precedence.

## Diagnostics Payload Mapping (Context Pressure/Compaction)

The following semantic primary fields are emitted by `core/runner` and parsed by `RuntimeRecorder` with legacy fallback:

- `context_pressure_zone` <-> `ca3_pressure_zone`
- `context_pressure_reason` <-> `ca3_pressure_reason`
- `context_pressure_trigger` <-> `ca3_pressure_trigger`
- `context_pressure_zone_residency_ms` <-> `ca3_zone_residency_ms`
- `context_pressure_trigger_counts` <-> `ca3_trigger_counts`
- `context_compaction_compression_ratio` <-> `ca3_compression_ratio`
- `context_spill_count` <-> `ca3_spill_count`
- `context_swap_back_count` <-> `ca3_swap_back_count`
- `context_compaction_mode` <-> `ca3_compaction_mode`
- `context_compaction_fallback` <-> `ca3_compaction_fallback`
- `context_compaction_fallback_reason` <-> `ca3_compaction_fallback_reason`
- `context_compaction_quality_score` <-> `ca3_compaction_quality_score`
- `context_compaction_quality_reason` <-> `ca3_compaction_quality_reason`
- `context_compaction_embedding_provider` <-> `ca3_compaction_embedding_provider`
- `context_compaction_embedding_similarity` <-> `ca3_compaction_embedding_similarity`
- `context_compaction_embedding_contribution` <-> `ca3_compaction_embedding_contribution`
- `context_compaction_embedding_status` <-> `ca3_compaction_embedding_status`
- `context_compaction_embedding_fallback_reason` <-> `ca3_compaction_embedding_fallback_reason`
- `context_compaction_reranker_used` <-> `ca3_compaction_reranker_used`
- `context_compaction_reranker_provider` <-> `ca3_compaction_reranker_provider`
- `context_compaction_reranker_model` <-> `ca3_compaction_reranker_model`
- `context_compaction_reranker_threshold_source` <-> `ca3_compaction_reranker_threshold_source`
- `context_compaction_reranker_threshold_hit` <-> `ca3_compaction_reranker_threshold_hit`
- `context_compaction_reranker_fallback_reason` <-> `ca3_compaction_reranker_fallback_reason`
- `context_compaction_reranker_profile_version` <-> `ca3_compaction_reranker_profile_version`
- `context_compaction_reranker_rollout_hit` <-> `ca3_compaction_reranker_rollout_hit`
- `context_compaction_reranker_threshold_drift` <-> `ca3_compaction_reranker_threshold_drift`
- `context_compaction_retained_evidence_count` <-> `ca3_compaction_retained_evidence_count`

## Compatibility Test Coverage

- Config parser compatibility:
  - semantic-only input
  - mixed semantic + legacy input (semantic precedence)
  - existing legacy-only coverage retained

- Diagnostics parser compatibility:
  - semantic-only `run.finished` payload path parsed into existing `RunRecord` fields
  - existing legacy-only tests retained
