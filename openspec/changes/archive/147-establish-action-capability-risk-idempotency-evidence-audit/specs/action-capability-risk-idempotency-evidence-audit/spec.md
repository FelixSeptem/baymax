## Purpose

为已准入的 Tool、MCP 和 Action 建立离线、确定性、可回放的风险与验收证据审计合同，使副作用、幂等、可逆性、前置条件及执行事实缺口能够被明确分类，而不会改变实际执行路径。

## ADDED Requirements

### Requirement: Audit input SHALL be a versioned admitted capability snapshot

审计输入 MUST 使用版本化的 `action_capability_audit.v1` 合同，并且只能包含宿主或既有准入 owner 已经归一化的 bounded capability snapshot。审计 MUST NOT 通过 registry、Tool、MCP、Provider、网络、文件系统、时钟、凭证或执行器动态发现能力。

#### Scenario: Valid admitted snapshot is accepted
- **WHEN** replay receives a versioned snapshot with stable capability identity, source, version or digest, and bounded action descriptors
- **THEN** replay accepts the input and produces a deterministic normalized audit projection without invoking any external dependency

#### Scenario: Unadmitted or dynamically discovered capability is rejected
- **WHEN** a fixture attempts to discover an action by calling a registry, tool, provider, network endpoint, or runtime executor
- **THEN** replay fails with a deterministic `action_capability_execution_or_discovery_detected` classification and produces no partial success

#### Scenario: Unsupported or malformed version fails fast
- **WHEN** the fixture version is unsupported or required snapshot identity fields are missing
- **THEN** replay fails with `action_capability_schema_drift` or `action_capability_unknown_version` and does not normalize a partial result

### Requirement: Action descriptors SHALL expose bounded risk and retry metadata

每个已审计 Action MUST 明确或显式声明 unknown 的 effect、副作用、risk level、reversibility、idempotency、preconditions、timeout/retry boundary、owner、scope 和 capability version/digest。缺失值 MUST NOT 被静默解释为 safe、idempotent、reversible 或 retryable。

#### Scenario: Complete descriptor is normalized
- **WHEN** an action descriptor contains bounded effect, side-effect, risk, reversibility, idempotency, precondition, timeout, retry, owner, scope, and version facts
- **THEN** the normalized projection preserves those facts and marks the descriptor eligible for evidence evaluation

#### Scenario: Missing idempotency or side-effect declaration is not treated as safe
- **WHEN** an action descriptor omits idempotency, side-effect, risk, or reversibility metadata
- **THEN** the audit emits a stable gap classification and does not mark the action compliant or safely retryable

#### Scenario: Unbounded precondition or retry metadata is rejected
- **WHEN** a descriptor contains unbounded text, raw command content, credentials, or an unbounded retry policy
- **THEN** replay fails with `action_capability_privacy_or_bound_violation` and does not truncate the value into a passing result

### Requirement: Audit SHALL distinguish intent, issued operation, and confirmed result evidence

审计 MUST distinguish Action intent, an operation that was issued, and an externally confirmed result. Evidence references MUST be bounded, correlation-preserving, and source-owned; an intent or accepted request MUST NOT be represented as a confirmed business result.

#### Scenario: Intent without issued evidence remains incomplete
- **WHEN** a fixture contains an Action intent and policy decision but no evidence that the operation was issued
- **THEN** the audit returns `insufficient_evidence` or an equivalent bounded gap and does not claim an external side effect

#### Scenario: Issued operation without confirmation is recoverable-unknown
- **WHEN** a fixture records that an operation was issued but the external result is missing or unknown
- **THEN** the audit identifies the operation as unconfirmed and requires query, compensation, or human review rather than authorizing blind retry

This applies to read-only and side-effecting operations alike: any `issued` fact without a correlated `confirmed` fact MUST NOT receive a `compliant` verdict.

#### Scenario: Confirmed result is correlated to the original action
- **WHEN** an issued operation has a bounded confirmation reference matching its action identity, attempt, target scope, and version
- **THEN** the audit accepts the confirmation relationship and exposes it as evidence without storing raw response content

#### Scenario: Confirmation has a different attempt or correlation
- **WHEN** issued and confirmed references do not share action identity, version, scope, attempt ID, and correlation ID
- **THEN** the audit MUST NOT count the operation as confirmed and MUST emit `action_capability_evidence_correlation_drift` or `insufficient_evidence`

### Requirement: Preview, approval, commit, and verification evidence SHALL be auditable

对于声明为高风险、不可逆或具有外部副作用的 Action，审计 MUST be able to determine whether Preview, Approve, Commit, and Verify stages are declared and evidenced. A missing stage MUST produce a gap or insufficient-evidence result; it MUST NOT be inferred from a successful tool or policy response.

#### Scenario: Complete four-stage evidence chain is compliant
- **WHEN** an action declares and references bounded Preview, Approve, Commit, and Verify stages with matching object version and authorization scope
- **THEN** the audit marks the stage chain complete and preserves the evidence references in normalized form

#### Scenario: Commit exists without Verify
- **WHEN** an external side-effect action has commit evidence but no bounded verification reference
- **THEN** the audit returns `action_capability_verify_evidence_missing` and does not mark the action delivery-complete

#### Scenario: Stage evidence records denial, timeout, failure, or unknown status
- **WHEN** a high-risk Preview/Approve/Commit/Verify record is present but its status is not a valid positive observation for that stage
- **THEN** the stage is not considered complete and the audit emits `action_capability_stage_evidence_insufficient` rather than `compliant`

#### Scenario: Approval scope changed before commit
- **WHEN** approval evidence targets a different action version, resource scope, identity, or risk context than the commit evidence
- **THEN** replay returns `action_capability_approval_scope_drift` and rejects the chain as compliant

### Requirement: Audit verdicts SHALL be deterministic and evidence-honest

审计 MUST emit exactly one normalized verdict from `compliant`、`gap`、`insufficient_evidence`、`not_applicable` for each action descriptor. Missing evidence, unknown external state, or conflicting observations MUST never be converted into `compliant` by default.

#### Scenario: Fully evidenced low-risk read action is compliant
- **WHEN** a read-only action has complete bounded metadata, no external side effect, and no conflicting observation
- **THEN** the audit emits `compliant` with the normalized digest and evidence sufficiency marker

#### Scenario: Conflicting declaration and observation is a gap
- **WHEN** declared idempotency, effect, risk, or execution-start facts conflict with bounded observed evidence
- **THEN** the audit emits `gap` with a stable `action_capability_declared_observed_drift` classification

#### Scenario: Evidence cannot support a conclusion
- **WHEN** required evidence is absent, redacted beyond correlation, or marked unknown
- **THEN** the audit emits `insufficient_evidence` and does not infer success, safety, reversibility, or rollback completion

### Requirement: Replay SHALL be offline, deterministic, bounded, and backward compatible

Replay MUST normalize equivalent input ordering identically, reject duplicate conflicting facts rather than using last-write-wins, enforce bounded sizes and privacy rules, and remain compatible with historical fixtures that omit the new audit fields through documented defaults. Replaying a fixture MUST NOT write RuntimeRecorder, diagnostics, snapshots, registries, or any source fact store.

当前 v1 bounds are: raw JSON at most 1 MiB, at most 64 cases/evidence/list entries per bounded list, at most 256 UTF-8 bytes per reference string, at most 256 bytes per summary, timeout between 0 and 24 hours, and retry maximum between 0 and 10. Sensitive markers and payload-like content are rejected, never truncated into passing evidence.

The parser MUST reject duplicate JSON object keys and unknown payload-like or secret-bearing field names such as payload, credential, password, reasoning, command, or response projections. Additive compatibility is limited to known optional v1 fields and historical fixtures that omit the entire audit extension; it does not allow arbitrary raw bodies to be silently ignored.

#### Scenario: Replaying the same fixture is idempotent
- **WHEN** the same valid fixture is replayed repeatedly
- **THEN** normalized output, verdict, digest, and drift classification remain identical and no source or counter state changes

#### Scenario: Duplicate conflicting evidence is rejected
- **WHEN** two records use the same stable evidence identity but disagree on stage, scope, result, or version
- **THEN** replay fails with `action_capability_duplicate_conflict` instead of choosing the last record

#### Scenario: Historical fixture remains readable
- **WHEN** an older lifecycle, action-gate, security, or timeline fixture omits the new audit projection
- **THEN** the fixture remains parseable with nullable/default behavior and is not reported as an audit failure solely because the extension is absent

#### Scenario: Unknown raw or duplicate JSON field is rejected
- **WHEN** an audit fixture contains a duplicate object key or an unknown payload-like, credential, reasoning, command, or response field
- **THEN** replay fails closed with a schema, duplicate-conflict, or privacy/bound classification and does not ignore the field

### Requirement: Run and Stream evidence projections SHALL remain semantically equivalent

当输入同时提供等价 Run/Stream action evidence projection 时，审计 MUST compare normalized effect, authorization, issued/confirmed state, verdict, and drift classification semantically. Byte-level event order MAY differ, but a semantic difference MUST be classified rather than hidden.

#### Scenario: Equivalent Run and Stream evidence passes parity
- **WHEN** equivalent Run and Stream projections contain the same action metadata and bounded evidence relationships
- **THEN** the audit reports parity with equivalent verdict and drift taxonomy

#### Scenario: Run and Stream confirmation differs
- **WHEN** one path contains confirmed external evidence and the other path remains unconfirmed for the same action identity
- **THEN** replay returns `action_capability_run_stream_evidence_parity_drift`

### Requirement: Audit gate SHALL preserve library-first ownership and fail closed on scope expansion

The audit gate MUST remain a library-first, offline verification layer. It MUST fail when the change introduces a transport listener, hosted execution/session store, credential store, dynamic capability download, global action queue, automatic compensation executor, second terminal state machine, or direct diagnostics writer.

#### Scenario: Contract-only audit passes library-first boundary
- **WHEN** implementation consumes bounded snapshots and emits replay/gate output without runtime execution or persistent control-plane state
- **THEN** the gate accepts the boundary and records no runtime behavior change

#### Scenario: Runtime or control-plane expansion is detected
- **WHEN** implementation adds one of the prohibited hosted, dynamic, transport, credential, queue, or alternate-terminal components
- **THEN** the gate fails with a deterministic `action_capability_library_first_boundary_violation` classification
