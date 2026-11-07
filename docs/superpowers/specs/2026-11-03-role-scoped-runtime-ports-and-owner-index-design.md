# Role-Scoped Runtime Ports and Owner Evidence Index Design

**Date:** 2026-11-03

## Decision

Deliver the evidence-index half of roadmap E now.  Do not change a runtime port
until a later proposal produces the required two-caller boundary-violation
evidence.  The present change is a static, versioned and read-only index that
makes every enrolled public semantic traceable to one owner, its OpenSpec main
specification, executable tests, replay evidence (or a concrete reason why
replay does not apply), a dedicated quality gate, public documentation and a
rollback/non-goal boundary.

The index contains separate entries for the two distinct semantics delivered by
archive 192 instead of treating its two owners as one ambiguous owner:

1. `resolver-material-bounded-admission`, owned by `context/hostprojection`.
2. `provider-neutral-model-surface-evidence`, owned by `model/conformance`.

It also contains `source-adapter-lifecycle`, owned by `host/sourceadapter`,
from archive 190.  These entries exercise bounded host admission, provider
conformance, and a durable source lifecycle without inventing a global owner.

## Canonical representation

`docs/owner-evidence-index.v1.json` is the single machine-readable source.  A
small schema carries `version` and `semantics`; each semantic has a stable `id`,
one `owner`, a `main_spec`, one or more test paths, a replay object, one
dedicated `gate`, one or more public `documentation` paths, and non-empty
`rollback_boundary` and `non_goals` text.  `replay.mode` is either `fixture`
with an existing fixture path or `not_applicable` with a concrete reason.

`docs/owner-evidence-index.md` explains the public reading model and links to
the canonical JSON.  It is intentionally a guide rather than a second copy of
the rows.

## Validation boundary

`tool/contributioncheck` parses the JSON with Go's standard library.  Its
validator rejects malformed JSON, an unsupported version, empty or duplicate
IDs, absent/multiple/ambiguous owners, missing structural paths, duplicate
entries, missing dedicated gates, gates not wired into each standard quality
path, a fixture replay without a fixture, and a replay exemption with no
reason.  It also compares each entry against `docs/documentation-ownership.md`
so the broad source-map row remains compatible with the exact semantic owner.

The validator reads only repository files.  It does not run a replay, execute a
gate, mutate the filesystem, reach a provider, or become a runtime registry.
The existing `contributioncheck` test package is the contract execution seam;
paired shell and PowerShell scripts run its focused tests and are themselves
wired into the existing quality gates.

## Documentation and status model

The documentation source map receives exact rows for bounded resolver material,
model-surface evidence and source-adapter lifecycle.  The main contract index
links to the canonical index rather than reproducing an unvalidated table.  The
README, docs index, module-boundary page and roadmap explain that the index is
evidence only: it never grants execution, authorization, recovery, effect or
diagnostic-write authority.

While the proposal is active, roadmap status names it as in progress.  On
archive it moves to archive 193 and the archive index becomes the status
authority.  The existing roadmap-status gate validates this state transition;
the owner-index validator additionally rejects enrolled main-spec paths that do
not exist.

## Non-goals and rollback

This change introduces no `core/types` port, `host.Coordinator` port, global
runtime registry, service locator, control plane, source store, provider
dependency in `context/*`, runtime config, diagnostics field or writer.  It
does not alter Run, Stream, terminal, recovery, effect or authorization
semantics.  A later port-narrowing proposal requires evidence that two distinct
callers can cross owner boundaries through the same broad interface.

Rollback removes this static index, its validator, the dedicated scripts and
the navigation/documentation references.  It has no persisted state or runtime
migration, and leaves all existing owner semantics unchanged.

## Risks and mitigations

- A formatted but misleading index could look authoritative.  Validation ties
  every reference to a repository path, verifies the dedicated gate wiring,
  and rejects ambiguous owners.
- A semantic with two real owner domains could be forced into a fake single
  owner.  Entries must be split; multiple owners are invalid in v1.
- Replay is not meaningful for every public semantic.  The explicit
  `not_applicable` mode requires a reason instead of fabricating a fixture.
- The index could drift after delivery.  It runs in the normal quality path and
  validates documentation-source-map owner compatibility on each run.
