## Purpose

定义一个有界、版本化、显式 opt-in 的宿主 context projection 接入合同，使宿主预过滤的来源引用可安全复用 Baymax 的装配、预算、压缩、handoff 与恢复 owner，而不转移业务授权、内容存储或业务提交职责。

## ADDED Requirements

### Requirement: Host context projection SHALL be versioned, bounded, and detached

The system SHALL accept an opt-in `host_context_projection.v1` descriptor only when it has a nonempty projection identity, a positive projection version, a canonical SHA-256 digest, an authority/capability digest, and one or more opaque source references. The descriptor SHALL be reference-only: it MUST NOT contain raw private-memory, knowledge-source, credential, tool-result, effect, or unfiltered transcript bodies. A descriptor MUST remain at or below 32 KiB serialized, contain no more than 64 references, use references no longer than 256 UTF-8 bytes and digests no longer than 128 UTF-8 bytes.

The system MUST reject an unknown version, malformed identity, nonpositive version, missing digest, empty/duplicate/oversized reference, invalid capability declaration, over-limit envelope, or post-admission mutable-input change before source resolution or context assembly. The accepted snapshot MUST be detached from caller-owned mutable maps and slices.

#### Scenario: Valid host projection is admitted without source side effects
- **WHEN** a host submits a schema-valid, bounded descriptor with matching canonical digest and opaque source references
- **THEN** the system accepts a detached projection snapshot without invoking a provider, tool, business effect, diagnostic writer, or source-body resolver during descriptor validation

#### Scenario: Invalid or oversized descriptor is rejected before resolution
- **WHEN** a descriptor has an unsupported schema, invalid digest, duplicate reference, missing required identity, or exceeds any declared bound
- **THEN** it is rejected with a bounded host-context classification before any source reference is resolved or context is injected

#### Scenario: Caller mutation does not alter accepted projection facts
- **WHEN** a caller mutates an input collection after an otherwise valid descriptor is accepted
- **THEN** the accepted identity, digest, references, visibility declarations, and capability facts remain unchanged

### Requirement: Host filtering and reference-first resolution SHALL preserve owner boundaries

The host SHALL decide role visibility, source trust, business-reference and tool allowlists, private-memory/knowledge access, and current business authorization before it supplies a projection descriptor. Baymax SHALL validate the descriptor and request content only through a host-owned resolver after successful validation; the resolver MUST re-check current source access before returning material for assembly.

The runtime MUST NOT infer access from a descriptor digest, persist resolved source bodies as a host context store, copy those bodies into handoff/replay/diagnostics, or add a Provider SDK dependency to `context/*`. Resolver denial, missing source, or expired source authorization MUST fail closed without partial context injection, source mutation, or business commit.

#### Scenario: Authorized source is resolved only after projection validation
- **WHEN** a valid descriptor references a source that the host currently authorizes
- **THEN** the resolver supplies only the currently authorized material to the existing assembly path after all descriptor checks have succeeded

#### Scenario: Source authorization changes after descriptor creation
- **WHEN** a descriptor remains syntactically valid but the host no longer authorizes one of its source references
- **THEN** resolution fails closed and no fragment from that descriptor is injected into the assembled model context

#### Scenario: Descriptor digest is not treated as authorization
- **WHEN** a caller presents a matching historical projection digest without current source authorization
- **THEN** the system rejects the projection and does not use the digest as evidence of access permission

### Requirement: Assembly SHALL retain source-owned budget and capability semantics

An admitted host projection SHALL enter the existing Context Assembler only after host filtering and reference resolution. Existing prefix/journal behavior, context pressure, compaction, and `budget_projection.v1` SHALL remain authoritative. The projection MAY consume the existing read-only budget view, but MUST NOT reserve budget, create a new counter, write back remaining budget, alter admission thresholds, or decide terminal outcomes.

For a child or delegated operation, declared capabilities MUST be equal to or a strict subset of the parent capability declaration. Any widening, missing parent evidence, or inconsistent plan correlation MUST reject before assembly. A host execution plan correlation, when provided, SHALL be checked as an immutable reference to the existing plan identity/version/digest contract and SHALL NOT recompile, authorize, or resume that plan.

#### Scenario: Budget pressure remains an existing derived view
- **WHEN** a valid host projection is assembled under incomplete or saturated source-owned budget facts
- **THEN** the existing derived-budget semantics determine the bounded context behavior and the host projection neither writes budget facts nor changes terminal behavior

#### Scenario: Child capability expansion is rejected
- **WHEN** a child projection declares a capability not present in its parent projection
- **THEN** the system rejects it before resolving references or starting child context assembly

#### Scenario: Optional plan correlation does not authorize execution
- **WHEN** a descriptor carries a valid existing host-plan identity and digest
- **THEN** the correlation is retained as evidence only and does not dispatch, resume, or authorize a plan

### Requirement: Handoff and restore SHALL revalidate host projection correlation

When a valid opt-in host projection reaches a legal existing handoff cut, the generated handoff SHALL carry an additive, nullable, bounded correlation containing the projection schema, identity/version/digest, capability digest, and opaque source-reference evidence. It MUST NOT contain resolved source bodies or business authorization payloads.

Before restore injects a correlated projection, the runtime SHALL complete its existing handoff schema/cut/reference validation, then validate projection correlation and require the host resolver to re-check current source authorization. Missing references, projection/digest drift, expired authorization, capability expansion, or resolver failure MUST leave the handoff un-restored for that projection and MUST NOT mutate source context, continue execution, or commit business effects. Restore remains distinct from replay and from business authorization/commit.

#### Scenario: Legal handoff retains bounded host projection correlation
- **WHEN** an admitted projection is compacted at a legal finalized, checkpoint, tool-finalized, or flushed-stream cut
- **THEN** the handoff carries only bounded correlation and opaque reference evidence needed for later revalidation

#### Scenario: Restore detects projection drift before injection
- **WHEN** a handoff's host projection identity, digest, capability subset, or referenced source no longer matches current authorized facts
- **THEN** restore fails closed before context injection and reports a bounded revalidation classification without exposing source bodies

#### Scenario: Restore does not turn handoff into business commit
- **WHEN** a correlated projection is successfully restored
- **THEN** it follows the existing assembler/reference-first path and does not itself authorize or execute a host business commit

### Requirement: Run and Stream conformance, offline replay, and public semantics SHALL remain aligned

Equivalent supported Run and Stream executions using equivalent valid host projections SHALL produce semantically equal normalized assembly admission, budget facts, handoff correlation, restore outcome, and bounded rejection classifications. The delivered contract SHALL include positive, negative, and exact-boundary unit coverage; real-runner integration with fake model/tool; a versioned, reference-only offline replay; and equivalent shell/PowerShell gates.

Replay MUST be deterministic and read-only. It MUST NOT call a host resolver, provider, tool, network, RuntimeRecorder, business store, or restore a live run. Replay and gates MUST classify at least invalid schema/bounds, source-reference loss, projection digest drift, capability expansion, unauthorized restore, body/privacy leakage, Run/Stream mismatch, and side-effect-before-rejection. Public API/schema, bounds, errors, owner boundaries, support scope, compatibility, and rollback documentation MUST be updated in the same change before implementation completion or archive.

#### Scenario: Equivalent Run and Stream paths agree
- **WHEN** equivalent valid projections are used by Run and Stream through the same supported source path
- **THEN** their normalized assembly, budget, handoff/restore correlation, and result classifications are semantically equal without fabricating stream callbacks or terminal outcomes

#### Scenario: Offline replay detects unsafe projection behavior
- **WHEN** a fixture declares projection drift, privacy leakage, unauthorized restore, or rejection after a source side effect
- **THEN** offline replay returns the stable drift classification without resolving a real source or changing runtime state

#### Scenario: Documented host context integration is delivered
- **WHEN** this capability is implemented
- **THEN** the offline example and maintenance documentation identify the semantic anchor, supported runtime path, rejection boundaries, expected markers, owner responsibilities, and rollback procedure
