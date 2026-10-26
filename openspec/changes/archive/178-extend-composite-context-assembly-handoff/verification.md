# Verification Record

## Scope

This record covers owner-bound `host_context_projection.v1` integration for
Composer child attempts, team pending local members, and workflow pending
steps. Default planbridge manifest context cells remain unsupported: owner API
evidence is not capability promotion.

Mailbox was audited at its actual owner path. `ResumeHostPlanContinuation`
claims an existing message delivery and projects the lifecycle outcome; it does
not invoke a host-plan handler or business effect. There is therefore no
message/delivery/lease/attempt-bound context execution safe point in this
change. The mailbox context replay case remains explicitly unsupported; no
planbridge executor, second consume loop, re-enqueue, or parent re-run was
introduced.

## Focused Evidence

- `go test ./context/hostprojection ./context/handoff ./context/assembler ./orchestration/composer ./orchestration/teams ./orchestration/workflow ./orchestration/mailbox ./orchestration/planbridge ./tool/diagnosticsreplay ./tool/contributioncheck -count=1`
  - Passed on 2026-10-25.
- `go test ./orchestration/teams -count=1`
  - Passed on 2026-10-25.
  - Covers pending-local-member binding, foreign rejection before resolver/claim,
    sibling isolation, vote aggregate continuity, deadline fence, fresh FileStore
    reopen/new Engine restore, Run/Stream parity, and remote binding rejection.
- `go test ./tool/diagnosticsreplay -count=1 -run TestReplayHostContextProjection`
  - Passed on 2026-10-25.
  - The versioned reference-only fixture includes workflow cross-step, team
    foreign-member, stale/expired/duplicate, privacy, authorization, and
    lifecycle-only mailbox-unsupported cases.
- `pwsh -NoProfile -File scripts/check-host-context-assembly-handoff-contract.ps1`
  - Passed owner suites through the contribution boundary on 2026-10-25.
- `go test ./tool/contributioncheck -count=1 -run 'TestHostContextProjectionGateScriptsAreEquivalentAndBlocking|TestHostContextProjectionOwnershipBoundary'`
  - Passed on 2026-10-25.

## Capability Source Projection Evidence

`CapabilityManifestWithHostContextEvidence` is the explicit source-owned
projection path for task 3.3. It accepts complete detached evidence from exactly
one owner and can promote only that form's `host_context_assembly` and
`host_context_handoff_restore` cells. The evidence binds owner namespace,
native-node scope, projection/continuation versions, build revision,
same-host/single-writer deployment fence, retention/expiry, resolver
reauthorization and independent evidence references. The constructor itself does
not resolve context, execute, claim, resume, or create a source.

The default manifest remains entirely context-unsupported. Existing child,
team and workflow owner integration evidence is intentionally not inferred into
that default. Mailbox remains explicitly unsupported with
`mailbox_handler_context_safe_point_unsupported`: its lifecycle continuation
projects an outcome, not handler/effect execution. Missing per-cell evidence,
foreign namespace, unsupported node/deployment and mailbox lifecycle-only facts
are rejected by manifest validation before plan admission.

- `go test ./orchestration/planbridge -count=1 -run 'TestCapabilityManifestWithHostContextEvidence|TestCapabilityManifestDeclaresContextDimensionsIndependently|TestDefaultMailboxContextCellsNameLifecycleOnlyLimitation'`
  - Passed on 2026-10-25 after a workspace-local Go cache was selected.

## Focused Delivery Verification

- `go test ./context/hostprojection ./context/handoff ./context/assembler ./core/runner ./orchestration/composer ./orchestration/teams ./orchestration/workflow ./orchestration/mailbox ./orchestration/planbridge ./tool/diagnosticsreplay ./tool/contributioncheck -count=1`
  - Passed on 2026-10-25.
- `go test ./tool/contributioncheck -count=1; go test -race ./orchestration/composer ./orchestration/teams ./orchestration/workflow ./orchestration/planbridge -count=1`
  - Passed on 2026-10-25.
- `pwsh -NoProfile -File scripts/check-host-context-assembly-handoff-contract.ps1`
  - Passed on 2026-10-25, including owner binding, deterministic replay, offline example and contribution boundary.
- `pwsh -NoProfile -File scripts/check-openspec-documentation-impact.ps1`
  - Passed on 2026-10-25.
- `pwsh -NoProfile -File scripts/check-openspec-example-impact-declaration.ps1`
  - Passed on 2026-10-25.
- `pwsh -NoProfile -File scripts/check-openspec-roadmap-status-consistency.ps1`
  - Passed on 2026-10-25.
- `openspec validate extend-composite-context-assembly-handoff --strict`
  - Passed on 2026-10-25.
- `openspec validate --all`
  - Passed on 2026-10-25: 147 passed, 0 failed. Existing repository warnings/infos are unchanged.
- `git diff --check`
  - Passed on 2026-10-25.

## Completion Lifecycle Boundary

The expanded-budget quality gate completed while this change was still listed
as in progress and returned `exit_code=0`, including the roadmap consistency
step. After task `5.3` was marked complete, `openspec list --json` correctly
changed the change status to `complete`. The roadmap now records the change as
`待归档` rather than `进行中`, so the
standalone roadmap and docs consistency checks pass without pretending that the
archive index has already changed. No archive or branch integration was
performed in this implementation continuation.

## Full Repository Verification

- `$env:GOCACHE='D:\\code\\baymax\\.gocache-composite-context'; go test ./...`
  - Passed independently on 2026-10-25.
- `$env:GOCACHE='D:\\code\\baymax\\.gocache-composite-context'; go test -race ./...`
  - Passed independently on 2026-10-25.
- `golangci-lint run --config .golangci.yml ./...`
  - Passed independently on 2026-10-25 with `0 issues` after using the required
    elevated cache access. The sandbox-only attempt was blocked by the host
    golangci cache permissions; no lint finding remained.
- `pwsh -NoProfile -File scripts/check-docs-consistency.ps1`
  - Passed on 2026-10-25.
- `pwsh -NoProfile -File scripts/check-quality-gate.ps1`
  - Passed with explicit `exit_code=0` on 2026-10-25 using
    `BAYMAX_QUALITY_GATE_TOTAL_TIMEOUT_SECONDS=3600` and
    `BAYMAX_QUALITY_GATE_STEP_TIMEOUT_SECONDS=1200`.
  - Completed all 88 steps in 2138.54 seconds, including repository hygiene,
    docs/example/roadmap governance, full `go test ./...`, full
    `go test -race ./...`, host-context assembly handoff, golangci-lint,
    performance, full-chain example smoke, and govulncheck.
  - The earlier default 1800-second invocation timed out at its internally
    repeated race step; that timeout is superseded by the captured successful
    expanded-budget run above.

## Environment Limits

All FileStore evidence is same-host, single-writer, sequential reopen only. It
does not prove multi-writer CAS, cross-machine coordination, distributed
cursor transactions, or a mailbox handler execution safe point.
