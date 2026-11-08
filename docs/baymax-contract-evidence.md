# Baymax Contract Evidence Matrix

**Baymax pin:** `v0.0.0-20261103091945-d2e08cd6dc8c`

This matrix is the Baymax-facing companion to the provenance record in
[upstream findings](upstream/baymax-findings.md). It separates execution
wiring, identity/aggregation, continuation restore, source replay and
projection reconciliation, and live provider smoke. A `verified` cell is only
valid inside the deployment assumptions in the row; `limited`, `not-verified`,
and `skipped-unconfigured` cells cannot promote a capability.

| Execution form / owner | Execution wiring | Identity / aggregation / terminal | Continuation restore | Source replay / projection | Provider live-smoke | Deployment condition |
| --- | --- | --- | --- | --- | --- | --- |
| direct runner | verified | verified | verified for native direct owner | limited, source-owner history only | skipped-unconfigured | explicit owner store for durable profile |
| child / composer | verified | verified | limited: child pause evidence is path-specific | limited: bounded child source cursor and retention only | skipped-unconfigured | explicit File owner + native history + executor; same-host single-writer sequential reopen |
| `team_parallel` | verified | verified | limited: team checkpoint/restore is owner/profile bound | limited: team source facts, no global cross-member order | skipped-unconfigured | explicit team owner; same-host single-writer sequential reopen |
| `team_vote` | verified | verified, including vote provenance/tie-break | limited: settled-member restore only; ambiguous in-flight is recovery-required | limited: bounded team source history | skipped-unconfigured | explicit team owner; same-host single-writer sequential reopen |
| workflow | verified | verified | verified for declared workflow profile | limited: source-owned replay and reconciliation | skipped-unconfigured | file-backed same-host single-writer sequential reopen |
| mailbox lifecycle | verified | verified for delivery/lease/ack/DLQ | not-verified as Agent continuation | verified for mailbox lifecycle replay only | skipped-unconfigured | mailbox remains lifecycle owner; Agent owner is separate |
| native cancellation / timeout | verified | verified at native source boundary | not-verified for all product control bindings | not-verified for cross-store atomic projection | skipped-unconfigured | native safe point plus source terminal evidence |
| native causal retry | verified | verified for source causation and immutable prior terminal | not-verified for product retry binding | limited; no uncertain effect replay | skipped-unconfigured | native source owner and effect idempotency owner remain separate |
| provider live paths | not-verified in this refresh | not-verified in this refresh | not-verified in this refresh | not-verified in this refresh | skipped-unconfigured | endpoint, model and key required |

## Evidence Dimensions

- **Execution wiring** proves that a form can be invoked and that failures are
  routed to the intended owner. It does not prove continuation.
- **Identity/aggregation/terminal** proves parent/child/member identity,
  aggregate or vote provenance, and terminal arbitration. It does not prove a
  fresh-process checkpoint restore.
- **Continuation restore** requires a persisted owner checkpoint, a compatible
  fresh process, pause-specific evidence for `input_required`,
  `approval_required`, and `budget_exhausted`, and no re-execution of a terminal
  effect.
- **Source replay/projection** requires source-owned stable identity and cursor,
  bounded replay, retention/gap handling and idempotent reconciliation. Source
  append and an external projection are separate non-atomic commits.
- **Provider live-smoke** requires an actual configured endpoint/model/key. The
  current status is `skipped-unconfigured`, not success and not an upstream bug.

## Source-Store and Deployment Limits

The owner must explicitly inject the source history store; no runner, child,
team, workflow or mailbox source creates an implicit default. Plain file history
is same-host single-writer sequential reopen. The fenced coordinator is
same-host multi-instance only. Cross-machine coordination is not verified, and
the source/projection boundary is non-atomic. If the source store cannot be
read or reconciled, the result is `recovery-required` or unavailable; the
system must not rerun an uncertain effect from the beginning.

Mailbox lifecycle replay has a separate cursor from an Agent event cursor. A
mailbox lease or redelivery does not claim an Agent checkpoint, and mailbox is
not an Agent continuation owner. Unsupported cells remain unsupported in the
planner; they must not be silently downgraded to direct.

## Historical Pin Rule

The older finding pin `v0.0.0-20261022053159-3d0ab5949790` is historical/legacy
provenance only. It must never be substituted for the current pin above when
declaring current capability evidence.
