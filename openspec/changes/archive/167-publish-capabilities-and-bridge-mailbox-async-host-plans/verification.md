# Verification record — Spec 1 mailbox async host plan bridge

Date: 2026-10-19 (Asia/Shanghai)

## Result

All required PowerShell, Go, lint, OpenSpec, documentation and dedicated
mailbox-plan conformance checks completed with exit code 0. The only
environmental limitation was the optional direct Bash invocation: this Windows
environment has no usable `/bin/bash` (`WSL ... execvpe(/bin/bash) failed: No
such file or directory`). The functionally equivalent PowerShell dedicated gate
ran successfully, and the equivalent `.sh` script is committed for Unix CI.

The dedicated gates intentionally execute an invalid replay fixture. Its inner
Go test fails with the expected replay-drift classification; the gate asserts
that failure and then exits 0. That is negative-contract evidence, not a gate
failure.

## Executed evidence

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./orchestration/planbridge ./orchestration/mailbox ./integration ./tool/diagnosticsreplay -count=1` | 0 | bounded manifest/binding, real owner, lifecycle and replay suites |
| `go test ./integration -run TestHostExecutionPlanMailboxAsync -count=1` | 0 | real owner duplicate/conflict/unavailable/version, lifecycle and Run/Stream parity |
| `pwsh -File scripts/check-mailbox-plan-bridge-contract.ps1` | 0 | PowerShell owner/lifecycle/parity/replay gate; intentional invalid fixture rejected |
| `pwsh -File scripts/check-host-execution-plan-contract.ps1` | 0 | existing bridge/example contract gate; intentional invalid fixture rejected |
| `go test ./...` | 0 | full repository test suite |
| `go test -race ./...` | 0 | full repository race suite |
| `golangci-lint run --config .golangci.yml` | 0 | repository lint |
| `pwsh -File scripts/check-docs-consistency.ps1` | 0 | documentation and roadmap consistency |
| `pwsh -File scripts/check-quality-gate.ps1` | 0 | full PowerShell quality path, including new mailbox gate |
| `openspec validate publish-capabilities-and-bridge-mailbox-async-host-plans --strict` | 0 | OpenSpec artifact validation |
| `pwsh -File scripts/check-openspec-documentation-impact.ps1` | 0 | public documentation commitment and paths |
| `pwsh -File scripts/check-openspec-example-impact-declaration.ps1` | 0 | example impact declaration |
| `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1` | 0 | active roadmap status consistency |
| `git diff --check` | 0 | whitespace/error-free diff |

## Risks and rollback evidence

- `mailbox_async` is explicit opt-in: a compatible manifest, non-nil existing
  mailbox owner and full host-plan binding are required. Missing/unavailable or
  incompatible inputs reject before publish and never fall back to direct Run.
- `Compile` snapshots the mailbox binding and derives the compiled plan digest;
  it cannot mutate the caller's binding, and a supplied mismatched digest
  rejects before source activation.
- Existing mailbox records remain authoritative. Rollback stops constructing the
  `MailboxBinding` / marks `mailbox_async` unavailable in the manifest; it does
  not delete pending messages, change retries, or reinterpret delivery/terminal
  evidence as a business commit.
- No durable source event history/cursor replay, cross-process checkpoint
  restore, or composite continuation is claimed. Those dimensions remain
  explicitly unsupported in the capability manifest for Spec 2/Spec 3.

## Outstanding environmental note

`scripts/check-mailbox-plan-bridge-contract.sh` could not be executed directly
on this host because `/bin/bash` is absent. Its commands and assertions mirror
the successfully executed PowerShell gate and it remains available to Unix CI.
