# Baymax Upstream Findings

This document is the provenance-facing record for upstream capability findings. It
must name the exact Baymax dependency pin used for the observation. A finding
from another pin is historical context only; it is not silently replayed as a
current-version result.

## Provenance

- **Current Baymax pin:** `v0.0.0-20261103091945-d2e08cd6dc8c`
- **Legacy finding pin:** `v0.0.0-20261022053159-3d0ab5949790`
- **Legacy status:** historical/legacy provenance only; it is not evidence that
  the current pin reproduces the same observation.
- **Evidence classes:** `contract-test` means deterministic repository evidence;
  `integration-test` means owner wiring and lifecycle evidence;
  `deployment-boundary` means the result is conditional on an explicit storage
  profile; `live-smoke` means a real provider request; and
  `skipped-unconfigured` means the live-smoke prerequisites were absent.

The current-pin record below is a conservative refresh. It does not promote a
capability merely because execution wiring, DTOs, callback history, or a direct
run exists.

## Current-Pin Findings

### Child/composer continuation

- **Current-pin status:** `limited`
- **Evidence class:** `contract-test`, `integration-test`,
  `deployment-boundary`
- **Deployment assumption:** explicit child source owner, File-backed owner
  store, native history and compatible executor on the same host; the verified
  profile is single-writer sequential reopen.
- **Observed result:** child execution identity, assignment, budget/effect
  boundary, source cursor, and `input_required`, `approval_required`, and
  `budget_exhausted` checkpoint paths are represented by the child contract.
  Fresh-process evidence exists only for the declared owner profile.
- **Unresolved boundary:** other owners, cross-machine restore, and any missing
  pause-specific evidence remain `unsupported`; an execution wiring result is
  not a complete continuation claim.
- **Capability consequence:** planner must keep an unverified child/fanout cell
  fail-closed and must not downgrade it to direct execution.

### Parallel/vote team continuation

- **Current-pin status:** `limited`
- **Evidence class:** `contract-test`, `integration-test`,
  `deployment-boundary`
- **Deployment assumption:** explicit team source owner and compatible File
  store on the same host; team restore is limited to the verified
  single-writer/sequential-reopen profile.
- **Observed result:** team, member and aggregate/vote checkpoint state retains
  settled members, pending members, failures, cancellation, participant
  provenance, aggregate policy and tie-break evidence. Terminal members are not
  re-executed on restore.
- **Unresolved boundary:** native pause/restore outside that owner profile,
  cross-machine coordination, and ambiguous in-flight members remain
  `unsupported` or `recovery-required`.
- **Capability consequence:** terminal execution evidence cannot promote
  `team_parallel` or `team_vote` beyond the declared profile, and planner must
  not silently fall back to direct.

### Source history and projection recovery

- **Current-pin status:** `limited`
- **Evidence class:** `contract-test`, `integration-test`,
  `deployment-boundary`
- **Deployment assumption:** the source owner explicitly injects an owner store.
  Plain FileStore supports same-host single-writer sequential reopen; the file
  coordinator supports same-host multi-instance fenced CAS. Neither declares
  cross-machine coordination.
- **Observed result:** source identity, stable event identity, source-local
  sequence, opaque cursor, retention floor, bounded `ReadAfter`, typed expired
  or gap errors, idempotent append and crash-before-projection reconciliation
  are available when the owner binding is present.
- **Unresolved boundary:** multi-writer cross-machine fencing and atomic commit
  between source history and an external projection are not verified. If the
  source store is unavailable, the owner must report `recovery-required` or
  unavailable rather than rerun uncertain effects.
- **Capability consequence:** source replay is read-only and bounded; it cannot
  be advertised as universal multi-process durable replay or as a
  source/projection transaction.

### Mailbox lifecycle

- **Current-pin status:** `limited`
- **Evidence class:** `contract-test`, `integration-test`
- **Deployment assumption:** the mailbox owns admission, lease, ack/requeue,
  retry, DLQ and lifecycle history. A separate runner, child, team or workflow
  owner must own Agent continuation.
- **Observed result:** lifecycle replay and a scheduled caller's logical
  admission are deterministic. Lease and delivery identity remain distinct
  from an execution checkpoint cursor.
- **Unresolved boundary:** mailbox replay is not Agent pause restore. There is no
  implicit mailbox-item-to-checkpoint binding, lease-transfer checkpoint
  authority, or terminal/ack atomicity contract for an Agent.
- **Capability consequence:** do not infer Agent continuation from redelivery,
  lease reclaim, callback history or a mailbox cursor. Any future Agent pause
  restore requires an explicit API and a separate finding.

### Native cancellation and timeout

- **Current-pin status:** `limited`
- **Evidence class:** `contract-test`, `integration-test`
- **Deployment assumption:** native owner safe points and source terminal
  evidence are available; adapter cancellation/timeout behavior is tested
  independently from product projection durability.
- **Observed result:** native cancellation and timeout causation, arbitration and
  terminal ordering are deterministic at the adapter boundary.
- **Unresolved boundary:** durable product control binding and projection of the
  product control decision remain pending; adapter evidence does not prove a
  cross-store atomic terminal commit.
- **Capability consequence:** a native cancellation/timeout result must not be
  presented as a fully durable product control binding without owner evidence.

### Native causal retry

- **Current-pin status:** `limited`
- **Evidence class:** `contract-test`, `integration-test`
- **Deployment assumption:** retry is admitted by the native source owner and
  retains causation and immutable prior terminal evidence.
- **Observed result:** causal retry facts and immutable prior terminal facts are
  deterministic at the native source boundary.
- **Unresolved boundary:** durable product retry binding and any external effect
  transaction are not verified. A source fact must not trigger an uncertain
  effect a second time.
- **Capability consequence:** native causal retry evidence does not by itself
  promote product-level retry or effect idempotency.

### Provider live-smoke

- **Current-pin status:** `skipped-unconfigured`
- **Evidence class:** `live-smoke`
- **Deployment assumption:** a real OpenAI-compatible endpoint, model and key
  must be supplied outside the repository. No credentials or endpoint are
  committed to fixtures or gates.
- **Observed result:** this refresh has no endpoint/model/key configuration, so
  the live-smoke is intentionally `skipped-unconfigured`.
- **Unresolved boundary:** no current-pin live provider request was observed;
  this status is neither a Baymax upstream bug nor a success claim.
- **Capability consequence:** offline direct/child/team/workflow/mailbox
  evidence cannot stand in for a real provider smoke. A configured run must
  separately cover those provider paths before any live claim is made.

## Interpretation Rules

The findings above preserve these negative claims: execution wiring is not
continuation restore; same-host evidence is not cross-machine support; source
history and an external projection are separate, non-atomic commits; mailbox
lifecycle replay is not an Agent continuation owner; and `skipped-unconfigured`
is not an upstream failure or a passing live test. Unsupported capability cells
remain fail-closed in the planner.
