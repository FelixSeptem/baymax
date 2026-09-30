## Purpose

为离线、确定性的 Agent 组合场景提供可版本化、可回放且有界的仿真与完成验证合同，明确运行完成、证据完整性、业务结果引用和发布准入之间的区别，并保持现有 Runtime、Run/Stream、completion 与 Outcome owner 不变。

## ADDED Requirements

### Requirement: Scenario profiles SHALL be versioned, bounded, and reference-first

仿真输入 MUST 使用稳定的场景标识和受支持的 schema version，并以有界引用描述用户、模型、工具、审批、取消、Stream 故障、恢复和预期证据。规范化 MUST 具备确定性；未知的 additive 字段 MUST 被安全忽略；缺失身份、重复身份、超出数量或大小上限的输入 MUST 在执行前失败。

#### Scenario: Valid scenario profile is normalized
- **WHEN** a scenario declares a supported version, stable identity, bounded actors/events, and valid source references
- **THEN** normalization emits a deterministic scenario digest and preserves the declared event and evidence ordering semantics

#### Scenario: Malformed scenario profile is rejected
- **WHEN** a scenario omits its version or identity, repeats an event identity, or exceeds a declared bound
- **THEN** validation fails with a deterministic scenario-schema classification and emits no partial run result

#### Scenario: Unknown additive field is ignored safely
- **WHEN** a scenario contains an unknown field that is outside the canonical contract and within the payload size bound
- **THEN** normalization succeeds without changing the canonical digest of known fields

### Requirement: Simulation execution SHALL be offline, deterministic, and test-support scoped

仿真 MUST 只消费受控的 fake model/SSE、tool、approval、cancellation、fault and recovery inputs，不得调用 live provider、网络、Git、workspace mutation、外部 artifact resolver 或生产级执行器。仿真 MUST 记录计划事件、命中事件、恢复验证和完成事件之间的 bounded causation，不得创建第二套 runtime event ordering、pending queue 或 terminal state machine。

#### Scenario: Controlled multi-step scenario executes offline
- **WHEN** a valid scenario supplies deterministic model, tool, approval, cancellation, truncation, and recovery inputs
- **THEN** simulation produces a bounded event trace without network or provider access and retains source-owned correlation identifiers

#### Scenario: Live dependency is attempted
- **WHEN** a simulation case requires a live provider, network endpoint, Git operation, workspace mutation, or hosted state lookup
- **THEN** the case is rejected with a deterministic offline-scope classification before side effects occur

#### Scenario: Planned fault does not imply observed fault
- **WHEN** a scenario declares a fault but the target event is not reached or the recovery assertion is not satisfied
- **THEN** the result records planned, targeted, and recovered statuses separately rather than treating the fault as observed success

### Requirement: Run results SHALL separate execution, evidence, business outcome, and release admission

每个仿真结果 MUST 提供四个独立且有界的结论：运行状态、证据完整性、业务结果引用和发布准入。证据不足、业务结果未提供或发布条件未满足时 MUST 返回 `indeterminate` 或对应的未满足分类，不能由模型自述、最终文本或单个 terminal 状态推断成功。

#### Scenario: Execution completes with sufficient evidence
- **WHEN** the controlled run reaches its authoritative terminal state and all required evidence references validate
- **THEN** the result reports execution completed, evidence complete, preserves the business outcome reference if supplied, and evaluates release admission independently

#### Scenario: Execution completes without business outcome evidence
- **WHEN** the runtime reaches a terminal state but the scenario lacks a valid business outcome reference or required verification evidence
- **THEN** the result reports execution completion while classifying business outcome or release admission as `indeterminate`

#### Scenario: Model claims success without verifier evidence
- **WHEN** model output says a task succeeded but required tool, policy, checkpoint, or side-effect evidence is absent or conflicting
- **THEN** the verifier rejects the success claim and emits a bounded evidence-incomplete classification

### Requirement: Completion verification SHALL be reference-only and source-owned

Verifier MUST validate evidence by bounded owner, stable identifier, digest/version and correlation references to existing event, checkpoint, artifact, policy, tool, memory, context, provider, host or runtime owners. Verifier MUST NOT copy or resolve raw reasoning, transcript text, provider responses, tool output bodies, memory/workspace content, credentials or unbounded payloads; it MUST NOT mutate runtime, task, plan, terminal, memory, experiment or feedback state.

#### Scenario: Valid evidence references are accepted
- **WHEN** required evidence references identify existing source-owned records with matching correlation and digest/version data
- **THEN** verifier marks evidence complete without copying the referenced body or changing the source owner

#### Scenario: Body-bearing evidence is rejected
- **WHEN** a result includes raw reasoning, transcript, provider response, tool output, memory, workspace, credential, or unbounded map content as evidence
- **THEN** verification fails with a deterministic privacy classification and leaves all source owners unchanged

#### Scenario: Conflicting evidence references are rejected
- **WHEN** the same owner and stable identifier appears with conflicting digest, version, or correlation data
- **THEN** verification emits a deterministic evidence-conflict classification and does not use last-write-wins behavior

### Requirement: Scenario replay SHALL be deterministic, idempotent, and drift-classified

Replay MUST accept versioned scenario and result fixtures without live runtime connectivity, normalize equivalent input ordering identically, and preserve expected and observed digests. Replaying the same case MUST be idempotent; conflicting duplicate case identities MUST fail instead of overwriting. Drift MUST distinguish schema, event-causation, evidence, completion, outcome, and admission differences.

#### Scenario: Equivalent fixture replay is stable
- **WHEN** the same scenario and result fixture is replayed repeatedly or with semantically equivalent ordering
- **THEN** replay produces the same normalized result digest and classification without duplicate side effects

#### Scenario: Replay detects semantic drift
- **WHEN** observed event causation, required evidence, completion status, business outcome reference, or release admission differs from the expected fixture
- **THEN** replay fails with the corresponding bounded drift classification and preserves expected and observed evidence

#### Scenario: Duplicate identity conflicts
- **WHEN** two cases use the same stable scenario/result identity with different canonical digests
- **THEN** replay returns a deterministic duplicate-conflict classification and does not apply last-write-wins

### Requirement: Run and Stream simulation SHALL preserve semantic equivalence

Equivalent scenario inputs, effective policy, evidence references, completion state and recovery state MUST produce equivalent Run and Stream execution, verification, terminal, idempotency and admission classifications after permitted event-order normalization. Stream-only incremental events MAY differ, but no parallel decision or termination semantic is allowed.

#### Scenario: Equivalent Run and Stream scenarios agree
- **WHEN** one scenario is exercised through Run and Stream with equivalent model, tool, approval, cancellation, truncation and recovery inputs
- **THEN** normalized execution, evidence, completion, terminal and release-admission conclusions are semantically equivalent

#### Scenario: Stream truncation is recovered once
- **WHEN** a Stream scenario truncates after a bounded event and recovery replays the pending completion reference
- **THEN** verifier records one recovered or not-applied outcome according to the existing safe-point policy, and Run/Stream classification remains equivalent

#### Scenario: Terminal or duplicate completion races
- **WHEN** a completion is delivered after terminal commit or more than once through either path
- **THEN** the result preserves the authoritative terminal outcome and reports late/duplicate classification without a second decision

### Requirement: Simulation outputs SHALL remain privacy-safe and compatibility-safe

Scenario, result, replay and diagnostic projections MUST use bounded identifiers, reason codes, digests, versions and nullable references. They MUST exclude secrets, raw payloads and unbounded bodies. Any diagnostic fields MUST be additive, nullable, default-compatible and written through the existing RuntimeRecorder single-writer path; historical payloads without simulation fields MUST remain readable.

#### Scenario: Historical payload is read without simulation fields
- **WHEN** replay or verification receives a pre-simulation diagnostic/result payload
- **THEN** parsing succeeds and simulation-specific fields resolve to documented nullable/default values

#### Scenario: Oversized output is submitted
- **WHEN** a scenario or result exceeds identifier, evidence, event, or serialized-size limits
- **THEN** validation fails or applies the existing bounded truncation policy with an explicit classification, without persisting an unbounded body

#### Scenario: Diagnostic projection is emitted
- **WHEN** a simulation result is correlated into diagnostics
- **THEN** the projection uses the existing RuntimeRecorder path, preserves Run/Stream correlation, and does not create a parallel writer or schema source of truth
