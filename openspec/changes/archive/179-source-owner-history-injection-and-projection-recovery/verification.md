# Verification: Source-Owned History Projection Recovery

## Scope

This verification covers the source-owner history injection and projection-recovery
change. The recovery path is reference-only: it consumes a retained source fact
and writes only the explicitly bound matching projection ledger. It does not create
an owner, queue, worker, scheduler, execution loop, or business side effect.

## Focused verification

The following focused suites passed with exit code `0`:

```text
go test ./core/history ./core/types ./core/runner ./orchestration/composer ./orchestration/teams ./orchestration/workflow ./orchestration/planbridge ./examples/native-execution-events
go test -race ./core/history ./core/types ./core/runner ./orchestration/composer ./orchestration/teams ./orchestration/workflow ./orchestration/planbridge ./examples/native-execution-events
```

The source-history projection PowerShell gate passed. Its contract coverage includes
retained source facts, fresh-process reconciliation, duplicate acknowledgement,
foreign/malformed/gapped/expired facts, unavailable stores, and the no-execution
side-effect boundary. The equivalent Bash gate is present, but cannot create its
process in this Windows sandbox (`Bash/Service/CreateInstance` permission failure).

The offline example passed:

```text
go run ./examples/native-execution-events
```

and emitted the expected `reconciled`, `duplicate`, `conflict`, and `unsupported`
markers.

## Repository-wide verification

These commands passed with exit code `0`:

```text
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml ./...
openspec validate source-owner-history-injection-and-projection-recovery --strict
openspec validate --all
pwsh -File scripts/check-docs-consistency.ps1
pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1
pwsh -File scripts/check-openspec-example-impact-declaration.ps1
pwsh -File scripts/check-openspec-documentation-impact.ps1
git diff --check
```

The lint command reported `0 issues`. It emitted only non-fatal cache permission
warnings from the user golangci cache. The full test commands were run with a
workspace-writable Go build cache and the existing module cache; an earlier attempt
with an empty module cache failed while downloading dependencies through the
restricted network, and the rerun with the existing cache passed.

The first full quality-gate invocation reached all contract suites and passed every
completed step, but its fixed 1,800-second total budget expired while
`go test -race ./...` still had work remaining. A second invocation used an explicit
3,000-second total budget and 900-second step budget. It completed `go test ./...`,
`go test -race ./...`, lint, all contract suites, and the A64 stability/performance
steps successfully. The final `govulncheck ./...` step returned exit `3` because
the repository scan reports 10 Go standard-library vulnerabilities, 6 imported
package vulnerabilities, and 1 required-module vulnerability. The scanner states
that the application does not appear to call those vulnerabilities. This is a
pre-existing repository security baseline issue, not a source-history regression;
it remains an explicit release risk and is not represented as a green quality-gate
result. The independent full test and race commands above passed.

## OpenSpec and delivery evidence

All planning artifacts validate strictly, and all 18 implementation tasks are
required to be checked before archive. The change's public semantic documentation
commitment and example impact assessment are synchronized across proposal, design,
tasks, README/integration/runtime-boundary/diagnostics/roadmap documentation, and
the native-events example.

## Fail-closed and rollback evidence

Recovery rejects malformed, foreign, gapped, expired, conflicting, unavailable,
and unbound inputs with typed errors before ledger mutation. A ledger persistence
failure leaves the journal fenced as `recovery-required`; recovery must be performed
against the same retained source fact and matching ledger before later appends can
proceed. Duplicate reconciliation is idempotent and returns an explicit duplicate
result. No recovery test invokes Run, Stream, retry, resume, restore, model, tool,
handler, lease, ack, nack, DLQ, business commit, or diagnostics writing. Rollback is
therefore fail-closed: removing the new reconciliation entry point leaves the
existing source journal and projection ledger contracts intact, while uncertain
projection state cannot silently trigger a second execution.
