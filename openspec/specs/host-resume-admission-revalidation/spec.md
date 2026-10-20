# host-resume-admission-revalidation Specification

## Purpose
This capability defines a bounded embedded-host revalidation contract before native dynamic-action resume, preserving source checkpoint ownership while preventing continuation under stale business authority, revision, or execution bindings.

## Requirements

### Requirement: Resume revalidation SHALL use trusted bounded host bindings

An explicitly enabled host-resume admission contract MUST use a versioned, bounded, reference-only host binding correlated with the authoritative paused Run/session/checkpoint, action and mode. It MUST identify the host plan version/digest, business revision, authority/tool/write-scope digests, confirmation reference and effect reference. The first profile MUST use version host_resume_admission.v1, references no larger than 256 UTF-8 bytes, digests no larger than 128 bytes, a new DTO serialized envelope no larger than 32 KiB and a positive plan version. The host MUST load trusted binding and current authorization facts; caller-declared digests, checkpoint metadata or replay MUST NOT substitute for current authorization. Missing, malformed, mismatched or oversized bindings and unsupported versions MUST fail closed before decision admission.

#### Scenario: Valid binding is revalidated against current facts
- **WHEN** the trusted paused binding matches the source checkpoint and the host authorizes the decision using current matching business facts
- **THEN** the source can admit that decision through its native checkpoint path without interpreting business payloads

#### Scenario: Request supplies a forged current digest
- **WHEN** a caller claims current authority or plan digests that differ from the host-owned binding or current policy
- **THEN** the resume is rejected without consuming the checkpoint or invoking provider/tool/business effect

#### Scenario: Enabled contract lacks a verifier or binding
- **WHEN** revalidation is explicitly required but its verifier or trusted binding is missing
- **THEN** admission fails closed and never falls back to unvalidated resume

#### Scenario: Invalid version or bounded field
- **WHEN** a binding has an unsupported version, an empty required identity, a nonpositive plan version or an over-limit reference/envelope
- **THEN** validation rejects deterministically without recording an accepted decision

### Requirement: Current host authority SHALL be checked before source resume commit

For a new confirm, deny or timeout admission, the host MUST revalidate decision authority, confirmation and effect-reference eligibility against current business facts. Confirm MUST also require the expected execution authority, revision, plan, write scope and tool capability bindings. Deny and timeout MUST use host-defined cancellation authority without implying permission to execute business writes. Verifier error, panic, cancellation or invalid result MUST reject. Rejection MUST preserve the source checkpoint, pending decision eligibility and existing result; it MUST NOT emit resolution/terminal events, advance iterations or execute provider/tool/effect.

#### Scenario: Business revision changes during pause
- **WHEN** current revision differs from the trusted execution binding for a confirm
- **THEN** admission reports revision drift and the Run remains paused without new effects

#### Scenario: Authority or execution binding is revoked
- **WHEN** the current authority, plan, write scope, capability, confirmation or effect eligibility fails revalidation
- **THEN** admission returns the appropriate bounded rejection and does not record the decision as received

#### Scenario: Invalid verifier outcome
- **WHEN** the verifier errors, panics, returns an invalid or mismatched response, or its request context is canceled before commit
- **THEN** the request is rejected without consuming the paused checkpoint

#### Scenario: Rejected request does not poison a later valid attempt
- **WHEN** an authority denial is followed by a new valid check for the same still-paused binding and idempotency key
- **THEN** the valid attempt can be admitted once through the native path

#### Scenario: Caller can deny but cannot execute
- **WHEN** a valid deny or timeout has current cancellation authority but lacks execution write authority
- **THEN** the source may admit cancellation according to host policy without executing business work

### Requirement: Source commit SHALL recheck races and preserve single continuation

The source MUST validate its authoritative checkpoint before revalidation and again immediately before recording the decision. Revalidation MUST NOT prevent source cancellation or checkpoint queries by holding a source lock across host code. Only one decision may commit for a checkpoint generation. Expiry, cancellation, replacement or an already-committed decision occurring during revalidation MUST be classified from the current source facts; no stale snapshot may overwrite a newer checkpoint or terminal outcome.

#### Scenario: Two confirms validate concurrently
- **WHEN** two decisions for one checkpoint pass host revalidation concurrently
- **THEN** only one decision commits and emits resolution or continues execution; the other receives an idempotent, in-progress or conflicting result according to full request identity

#### Scenario: Cancellation or expiry wins during validation
- **WHEN** source cancellation commits or the checkpoint expires before the decision commit boundary
- **THEN** resume rejects without adding resolution, continuation or a synthetic terminal event

#### Scenario: Continuation pauses for another action
- **WHEN** an admitted continuation produces a new PendingAction and an old decision is submitted again
- **THEN** the old request cannot consume or overwrite the new paused checkpoint

### Requirement: Duplicate requests SHALL preserve full identity and result-read boundaries

Duplicate equivalence MUST include normalized Run/session, checkpoint identity/version/digest, action token, decision kind/reference, mode and idempotency key. A completed exact duplicate for a source-retained admission MUST return its existing result without a fresh business revalidation, resolution, terminal emission or continuation. An in-flight duplicate MUST report a stable in-progress classification without representing an empty result as completion. Non-equivalent requests MUST NOT be accepted as duplicates. The host MUST authorize result access independently; idempotency identity is not caller authentication.

#### Scenario: Completed confirm is repeated exactly
- **WHEN** the completed decision is repeated with the full identical normalized identity while its admission is retained
- **THEN** the prior source result is returned and verifier, resolution and effect counts do not increase

#### Scenario: Key and kind match but checkpoint or reference differs
- **WHEN** a request reuses a key and decision kind with a different session, checkpoint, digest, token, mode or decision reference
- **THEN** it is rejected as mismatched or conflicting and never returns duplicate success

#### Scenario: First continuation is still executing
- **WHEN** an exact duplicate arrives before the first continuation result is available
- **THEN** it reports in-progress and does not execute a second continuation

### Requirement: Resume support SHALL distinguish live continuation from metadata recovery

The runtime MUST preserve same-Engine direct dynamic-action Run/Stream resume using their original modes. It SHALL additionally support new-Engine direct restore only through a configured compatible durable checkpoint profile containing actual source continuation state and an atomic source resume claim. Snapshot references, metadata-only checkpoint interfaces, shared Run identifiers and event replay MUST NOT imply durable runner continuation. Plan bridge resumed execution and composed child/team/workflow/mailbox continuations MUST retain their unsupported classifications in this profile. Resume MUST remain distinct from retry, new Run and event replay.

#### Scenario: Live Run and Stream resume
- **WHEN** equivalent active in-process checkpoints receive equivalent admitted decisions in their respective modes
- **THEN** source iteration, saved tool results, resolution and terminal semantics remain equivalent

#### Scenario: Process restarts with metadata only
- **WHEN** a new Engine receives a decision and previous checkpoint metadata without retained execution continuation
- **THEN** it reports unavailable or unknown without invoking provider/tool or synthesizing a new Run

#### Scenario: Plan trigger is labeled resumed
- **WHEN** a planbridge descriptor uses resumed delivery without a supported native bridge continuation path
- **THEN** the existing unsupported result remains and the label is not treated as recovery authorization

#### Scenario: Process restarts with a compatible durable source
- **WHEN** a new Engine loads the original direct checkpoint from a compatible durable store and current host revalidation and source claim succeed
- **THEN** it restores the same Run/session and saved continuation boundary without a new parent Run or completed-tool reexecution

### Requirement: Revalidation SHALL preserve source events and offline conformance

Accepted confirm, deny and timeout MUST preserve existing single resolution semantics; only deny/timeout produce canceled terminal. Rejection and duplicates MUST NOT introduce resolution or terminal events. Bounded admission classifications MUST use existing event/RuntimeRecorder paths, excluding business payloads, secrets and raw callback errors. Versioned offline fixtures MUST cover legal resume, drift, mismatch, duplicates, races, process recovery scope and Run/Stream parity. Equivalent shell/PowerShell gates and synchronized public documentation MUST verify these contracts without adding an execution, effect, terminal or diagnostics owner.

#### Scenario: Accepted decisions retain native event semantics
- **WHEN** confirm, deny or timeout is admitted after revalidation
- **THEN** exactly one resolved event is emitted and terminal behavior follows the original native decision semantics

#### Scenario: Offline fixture detects semantic drift
- **WHEN** replay observes rejection with a resolution/effect, a duplicate continuation or Run/Stream admission divergence
- **THEN** it reports contract drift without executing host callbacks, providers or tools

#### Scenario: Public resume semantics change during implementation
- **WHEN** public fields, bounds, errors, admission order or support scope change
- **THEN** the same change updates maintenance documentation and the governance gate verifies the declared paths and evidence

### Requirement: Host revalidation SHALL precede durable resume claim

When host resume admission is enabled for a durable checkpoint, the verifier SHALL validate the current reference-only authority, revision, plan, scope, confirmation, and effect binding before the source records a resume claim. The verifier SHALL never load or reconstruct source continuation and SHALL run outside source locks; source identity and generation MUST be rechecked before claim commit.

#### Scenario: Revalidation denial preserves checkpoint
- **WHEN** current host authority or revision no longer matches the trusted binding
- **THEN** admission rejects without consuming the checkpoint, emitting resolution/terminal events, or invoking provider/tool/effect

#### Scenario: New Engine passes valid revalidation
- **WHEN** a reopened source checkpoint and current host facts match
- **THEN** exactly one durable claim is committed and continuation retains the original run/session identity
