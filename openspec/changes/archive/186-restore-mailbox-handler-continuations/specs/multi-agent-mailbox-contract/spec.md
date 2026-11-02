## ADDED Requirements

### Requirement: Mailbox worker handler pause SHALL be an owner-native safe boundary

An explicitly configured mailbox worker SHALL be able to report
`input_required`, `approval_required`, or `budget_exhausted` at a declared
handler effect-safe boundary. The pause request SHALL contain only bounded
reason, pending-action/effect reference, completed-effect references, consumed
budget, original deadline, expiry, and handler checkpoint material. The mailbox
owner SHALL atomically persist that material with the original host-plan
identity, message, delivery, lease, attempt causation, and lifecycle facts.
Payload bodies, credentials, callback errors, and host business-commit claims
MUST NOT be retained in that material.

The pause transition MUST retain the original in-flight delivery and MUST NOT
acknowledge, negative-acknowledge, requeue, retry, publish, consume a
replacement delivery, or create a parent Run. A worker without this explicit
handler continuation configuration SHALL retain existing handler-error and
lifecycle behavior and MUST NOT claim a handler safe point.

#### Scenario: Handler pause retains the original delivery
- **WHEN** a configured handler returns a valid approval pause before its
  pending effect is dispatched
- **THEN** the mailbox persists exactly one continuation bound to the current
  message, delivery, lease, and attempt causation
- **AND** the record remains the original in-flight delivery without an Ack,
  Nack, Requeue, retry, or replacement consume

#### Scenario: Unconfigured worker does not infer a pause boundary
- **WHEN** a worker has no configured continuation handler/resumer pair
- **THEN** a requested host-handler pause returns the stable typed unsupported
  outcome before handler/effect dispatch
- **AND** lifecycle continuation DTOs and history replay do not enable it

### Requirement: Mailbox handler continuation restore SHALL atomically claim and resume once

The mailbox owner SHALL expose an explicit handler-continuation restore entry
that validates schema/digest, owner namespace, durable deployment scope,
original host-plan identity, message/delivery/lease/attempt causation, pause
reason, deadline/expiry, authority and current resolver reauthorization before
claiming. It SHALL atomically retain the request identity and invoke the
registered source-owned resumer at most once for that claim.

Equivalent duplicate requests SHALL return the retained in-progress or terminal
projection. Foreign, malformed, stale, expired, conflicting, unavailable,
lease-lost, reauthorization-denied, and ambiguous claimed records SHALL return
their typed failure before a handler/effect fallback. The restore entry MUST NOT
re-enqueue, re-consume, publish a replacement message, or re-run a parent.

#### Scenario: Fresh process resumes an original handler boundary
- **WHEN** a compatible new process reopens the configured owner state and
  submits the same valid handler continuation request
- **THEN** the mailbox validates and claims the original delivery lineage and
  invokes its registered resumer once
- **AND** completed effects remain references and are not re-executed

#### Scenario: Duplicate restore converges without a second handler invocation
- **WHEN** the same restore request repeats after a claim or terminal result
- **THEN** the owner returns the retained claim/result projection
- **AND** it does not invoke the resumer, publish, consume, or create a second
  logical execution

### Requirement: Mailbox handler continuation terminal effects SHALL remain lifecycle-owned

After a claimed resumer completes, the mailbox owner SHALL apply its existing
Ack, Nack, Requeue, retry, DLQ, timeout, cancellation, and lease-loss rules to
the original delivery. The resulting public projection SHALL preserve message,
delivery, lease, attempt/retry causation, run/parent run, host correlation,
authority, occurrence, effect namespace, lifecycle result, and terminal
outcome. It MUST remain distinct from a host business commit.

An expired or lost lease, owner persistence failure, or process crash between
claim/start/terminal persistence MUST return source evidence or
`recovery-required`; it MUST NOT synthesize retry, completion, handler restart,
or parent Run.

#### Scenario: Resumer failure follows the original mailbox policy
- **WHEN** a claimed resumer returns a recoverable handler failure
- **THEN** the mailbox applies the configured lifecycle policy to the original
  delivery and emits its retry or DLQ terminal evidence
- **AND** it preserves the attempt causation chain without asserting a business
  commit

#### Scenario: Lease loss prevents handler restart
- **WHEN** the original delivery lease is reclaimed after pause persistence and
  before or during resume
- **THEN** the restore returns typed lease-loss or recovery-required evidence
- **AND** no resumer, replacement consume, retry, or parent Run is started
