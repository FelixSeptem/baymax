## 1. Protocol object model and lifecycle contract

- [x] 1.1 Inventory existing `RunID`, `SessionID`, ToolCall, workflow step, teams task, scheduler attempt, A2A Task, realtime envelope, handoff artifact, snapshot manifest, timeline, and trace correlation fields; publish the one-way ownership/mapping table in the design and runtime documentation.
- [x] 1.2 Add failing `core/types` tests for `SessionRef`, `RunRef`, `StepRef`, `EventEnvelope`, `ArtifactRef`, and `CheckpointRef` validation, including required IDs, optional correlations, and stable JSON serialization.
- [x] 1.3 Implement the additive reference-only protocol DTOs and pure validators in `core/types`; run focused tests and `gofmt`.
- [x] 1.4 Add failing `core/types` state-machine tests for legal `submitted -> working -> input_required -> working -> terminal` transitions, idempotent cancel, resume rejection outside `input_required`, retry causation, and terminal-state immutability.
- [x] 1.5 Implement the minimal Run lifecycle state machine and deterministic error classifications without changing existing Runner, Scheduler, A2A, or realtime source states.
- [x] 1.6 Add negative tests proving configuration, security/permission, protocol validation, snapshot compatibility, and module-boundary failures cannot be converted into recoverable Step outcomes.

## 2. Runtime and orchestration projection adapters

- [x] 2.1 Add failing Runner tests that map equivalent Run and Stream executions to semantically equivalent protocol Run/Step outcomes after permitted event-order normalization.
- [x] 2.2 Implement Runner mapping adapters for model iteration, tool call, final outcome, recoverable tool failure, and terminal error while keeping `core/runner` as execution owner.
- [x] 2.3 Add failing Workflow, Teams, and Scheduler tests for mapping workflow steps, team tasks, scheduler attempts, retries, and parent-child correlations without changing their source state machines.
- [x] 2.4 Implement orchestration mapping adapters in the owning packages and preserve workflow/team/task/attempt source metadata.
- [x] 2.5 Add failing A2A tests for canonical Run/Step/Event/Artifact mapping of submitted, running, terminal, and correlated remote tasks.
- [x] 2.6 Implement A2A mapping adapters that retain Agent Card/task ownership, transport behavior, and supplied `workflow_id`, `team_id`, `task_id`, `step_id`, `agent_id`, and `peer_id` fields.
- [x] 2.7 Add failing realtime tests for interrupt-to-`input_required`, valid resume, invalid cursor/terminal resume, monotonic sequence, and duplicate-event mapping parity.
- [x] 2.8 Implement realtime-to-protocol adapters by reusing `realtime_event_protocol.v1` event IDs, sequence, deduplication, cursor, and existing Run/Stream parity behavior.

## 3. Event, artifact, checkpoint, and observability lineage

- [x] 3.1 Add failing protocol event-mapping tests for `types.Event`, timeline, realtime, and A2A sources, including source metadata, `event_id`, timestamp, `run_id`, `step_id`, and `causation_id` handling.
- [x] 3.2 Implement a canonical event mapper that preserves realtime ordering only for realtime sources and never assigns synthetic realtime sequence numbers to other source events.
- [x] 3.3 Add failing RuntimeRecorder and diagnostics tests proving protocol correlation fields are additive, nullable, replay-idempotent, and still written solely via standard events and `RuntimeRecorder`.
- [x] 3.4 Implement additive diagnostics/event mapping for protocol correlation and source metadata without adding direct diagnostics-store writers outside `RuntimeRecorder`.
- [x] 3.5 Add failing OTel semantic-convention tests for protocol `run_id`, `step_id`, source metadata, and available artifact/checkpoint lineage on run/model/tool/MCP/HITL spans.
- [x] 3.6 Extend `observability/trace` mapping and tests while preserving existing canonical topology, attributes, exporter behavior, and tracing/eval replay semantics.
- [x] 3.7 Add failing artifact/checkpoint tests for isolate-handoff reference projection and snapshot-manifest reference projection, including no artifact-body duplication, digest preservation, and source Run/Session correlation.
- [x] 3.8 Implement `ArtifactRef` and `CheckpointRef` adapters without creating content storage or changing `strict|compatible` snapshot import, compatibility-window, segment-owner, or idempotency behavior.

## 4. Contract replay and quality-gate coverage

- [x] 4.1 Define `agent_runtime_protocol.v1` replay fixture schema and canonical normalization rules for lifecycle, correlation, source mapping, event ordering/deduplication, artifact/checkpoint lineage, and failure boundary results.
- [x] 4.2 Add success, invalid-transition, missing-correlation, source-mapping-drift, realtime-ordering/dedup-drift, lineage-drift, and control-plane-absence fixtures under `integration/testdata/diagnostics-replay/`.
- [x] 4.3 Add `tool/diagnosticsreplay` parser/normalizer tests and integration replay tests that assert deterministic classification for each fixture outcome.
- [x] 4.4 Add `scripts/check-agent-runtime-protocol-contract.sh` and `scripts/check-agent-runtime-protocol-contract.ps1`; ensure both run the same required suites and classify fixture or control-plane failures identically.
- [x] 4.5 Add gate tests for pass/fail cases, update `scripts/check-quality-gate.sh/.ps1`, and register the dedicated CI required-check candidate.
- [x] 4.6 Run focused protocol, replay, Runner, orchestration, A2A, snapshot, tracing, and gate suites; repair all Run/Stream, import/replay, and shell/PowerShell parity regressions.

## 5. Documentation and example delivery

- [x] 5.1 Update `README.md` and `docs/runtime-module-boundaries.md` with the Protocol-versus-Runtime boundary, module ownership, and explicit library-first/non-control-plane constraints.
- [x] 5.2 Update `docs/runtime-config-diagnostics.md` with protocol object/state/event/lineage fields, error boundary, replay fixture, drift taxonomy, and gate execution details.
- [x] 5.3 Update `docs/mainline-contract-test-index.md` with protocol unit, integration, replay, gate, and CI mappings; update `docs/development-roadmap.md` status and evidence as work completes.
- [x] 5.4 Before code, add the `examples/agent-modes` matrix and README baseline for a real Agent Runtime Protocol projection example, including semantic anchor, runtime path, expected markers, and rollback notes.
- [x] 5.5 Implement the example only after its documentation baseline passes the doc-first gate; demonstrate Run/Step/Event/Checkpoint/Artifact reference correlation through existing runtime paths rather than synthetic markers.
- [x] 5.6 Add example smoke and integration coverage for both positive mapping and a deterministic rejected transition or correlation failure.

## 6. Full verification and OpenSpec closeout

### Example Impact Assessment

- 新增示例

- [x] 6.1 Run `go test ./...` and resolve all failures.
- [x] 6.2 Run `go test -race ./...` and resolve all data races or failures.
- [x] 6.3 Run `golangci-lint run --config .golangci.yml` and resolve all findings.
- [x] 6.4 Run `pwsh -File scripts/check-agent-runtime-protocol-contract.ps1`, `pwsh -File scripts/check-quality-gate.ps1`, and `pwsh -File scripts/check-docs-consistency.ps1`; record all verification evidence and any intentionally unexecuted item with reason. Protocol gate and docs consistency passed; quality gate completed through all code/contract/performance stages but failed at existing `govulncheck` findings (17 reachable dependency/standard-library vulnerabilities across 3 modules and Go 1.26.1).
- [x] 6.5 Reconcile proposal, design, specs, tasks, docs, tests, fixtures, scripts, and examples; confirm Example Impact Assessment evidence before checking tasks complete. `proposal.md`, `design.md`, and `tasks.md` declare `新增示例`; docs, fixtures, scripts, examples, MATRIX, PLAYBOOK, and CI mappings are present and consistency gates pass.
- [x] 6.6 Archive only after all tasks are complete by running `pwsh -File scripts/openspec-archive-seq.ps1 -ChangeName "introduce-agent-runtime-protocol-contract"` and update the archive index through the governed workflow.
