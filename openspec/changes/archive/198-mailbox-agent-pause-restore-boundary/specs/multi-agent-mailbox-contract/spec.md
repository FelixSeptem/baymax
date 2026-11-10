## Added Requirements

### Requirement: Configured mailbox handler continuation SHALL expose a bounded public capability

When a mailbox owner is configured with `NewContinuationWorker`, a durable
`FileStore`, a registered resumer, and a current revalidator, it SHALL declare
handler execution safe-point support only within the owner-reported
`same_host_single_writer_sequential_reopen` deployment scope. The owner SHALL
declare `input_required`, `approval_required`, and `budget_exhausted` separately;
each supported cell MUST reference evidence for fresh-process restore,
lease/revalidation admission, terminal projection, duplicate claim convergence,
and completed-effect non-replay. A default `NewWorker`, memory store, lifecycle
history reader, or host-plan lifecycle continuation MUST NOT promote these
cells.

#### Scenario: Configured owner promotes all evidenced pause cells

- **WHEN** a durable continuation worker exposes complete evidence for the
  three canonical handler pause reasons
- **THEN** the mailbox capability declares handler safe point and all three
  pause cells supported with the same bounded deployment scope

#### Scenario: Lifecycle-only owner keeps handler cells unsupported

- **WHEN** a mailbox has only its normal worker, lifecycle continuation, or
  retained history configured
- **THEN** all three handler pause cells remain unsupported with
  `mailbox_handler_safe_point_unsupported`

### Requirement: Handler continuation SHALL preserve successor pause lineage

If a registered resumer reaches another valid handler pause, the mailbox owner
MUST persist a successor continuation bound to the same message/delivery/lease
and incremented generation, with predecessor and successor references. The
owner MUST retain completed-effect references and MUST NOT invoke the original
handler or repeat completed effects. Duplicate requests for either the prior
terminal projection or the successor continuation MUST converge to the retained
source projection.

#### Scenario: Resumer pause creates a successor without rerunning effects

- **WHEN** a fresh process resumes a valid continuation and the resumer returns
  a valid pause for another canonical reason
- **THEN** the owner persists one successor checkpoint with lineage and returns
  a source pause projection
- **AND** the original handler/effect counter does not increase

#### Scenario: Successor duplicate resumes once

- **WHEN** the same successor resume request is submitted repeatedly
- **THEN** exactly one successor resumer invocation is admitted and all callers
  observe the retained terminal or pause projection
