## ADDED Requirements

### Requirement: Workflow continuation restore SHALL use an explicit owner binding

The host execution plan bridge SHALL accept workflow continuation restore only when the binding supplies an explicit source-owned resume request and a compatible workflow continuation owner/store. It MUST validate binding identity, manifest compatibility, requested capability cells, and deployment scope before checkpoint claim or step dispatch. The bridge MUST delegate restore to the existing workflow owner and preserve its native result. Run and Stream MUST share the same restore decision and terminal semantics; Stream may additionally deliver native owner events.

#### Scenario: Workflow restore delegates without replaying Run
- **WHEN** a compiled workflow plan is executed with an owner-bound continuation request
- **THEN** Run and Stream invoke the workflow owner's resume entry with the same request
- **AND** neither invokes normal Run/Stream to recreate the graph or reruns completed steps

#### Scenario: Missing continuation store is rejected before effects
- **WHEN** a workflow resume request is supplied without a compatible continuation store or required manifest evidence
- **THEN** the bridge returns typed unavailable or `unsupported_capability` before claiming a checkpoint or dispatching a step

#### Scenario: Foreign or mismatched resume identity is rejected
- **WHEN** the resume request does not match the compiled workflow/run/checkpoint identity, authority, plan, write scope, effect namespace, occurrence, or declared deployment scope
- **THEN** the bridge fails closed with a typed binding or owner error
- **AND** it does not activate workflow steps or fall back to normal Run/Stream

### Requirement: Plan bridge continuation capability SHALL be form and owner scoped

The bridge MUST report continuation support only for the configured source owner and store whose evidence matches the requested execution form, deployment profile, and pause reason. A sibling form's owner, package presence, or continuation DTO MUST NOT promote the capability. Forms without matching execution-safe-point and restore evidence MUST remain explicitly unsupported or unavailable.

#### Scenario: Sibling owner evidence does not promote workflow
- **WHEN** a workflow plan is bound only to child or team continuation evidence
- **THEN** the workflow continuation capability remains unsupported and activation is rejected before step dispatch

#### Scenario: Mailbox lifecycle support does not imply handler restore
- **WHEN** a mailbox owner exposes lifecycle admission/projection but not a handler execution safe point
- **THEN** mailbox handler continuation remains unsupported before publish, claim, or handler dispatch
