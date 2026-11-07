# Verification: role-scoped-runtime-ports-and-owner-index

## Scope review

The change adds a static, repository-relative owner/evidence index and offline
governance checks only. It does not add a runtime registry, service locator,
runtime port, source owner, authorization/recovery/effect control, diagnostic
writer, provider dependency, configuration key, or execution path. The three
enrolled semantics remain separately owned by `context/hostprojection`,
`model/conformance`, and `host/sourceadapter`.

## Commands and results

| Command | Result |
| --- | --- |
| `go test ./tool/contributioncheck -run 'Test(ValidateOwnerEvidenceIndex|OwnerEvidenceIndex)' -count=1` | pass |
| `pwsh -File scripts/check-owner-evidence-index-contract.ps1` | pass |
| `pwsh -File scripts/check-docs-consistency.ps1` | pass |
| `pwsh -File scripts/check-openspec-documentation-impact.ps1` | pass |
| `pwsh -File scripts/check-openspec-example-impact-declaration.ps1` | pass |
| `pwsh -File scripts/check-openspec-roadmap-status-consistency.ps1` | pass |
| `go test ./...` | pass; all packages, including `tool/contributioncheck`, passed |
| `go test -race ./...` | pass; all packages passed |
| `golangci-lint run --config .golangci.yml` | pass; 0 issues |
| `pwsh -File scripts/check-quality-gate.ps1` | pass; 93/93 steps, including the owner evidence gate, full tests, race, lint, A64 and govulncheck |
| `openspec validate role-scoped-runtime-ports-and-owner-index --strict` | pass |
| `openspec validate --all` | pass; 152 passed, 0 failed |
| `git diff --check` | pass; only line-ending warnings from Git on Windows |

The full quality gate also reported no vulnerabilities reachable by this code.
Historical OpenSpec warnings about placeholder Purpose sections remain outside
this change and do not cause validation failure.

## Shell parity limitation

The paired shell scripts are present, fail-fast, and covered by
`TestOwnerEvidenceIndexGateScriptsStayPairedAndWired`; both quality scripts
name and invoke the shell gate. Direct execution of either shell script was
attempted with `C:\Program Files\Git\bin\bash.exe` and could not start in this
Windows environment: Git Bash failed before script launch with `Win32 error 5`
(`couldn't create signal pipe` / `CreateFileMapping ... Access is denied`). No
passing Shell execution is claimed. PowerShell execution and the full blocking
quality path passed.

## Risks and rollback

The main risk is stale or ambiguous evidence producing false confidence. The
validator fails closed on malformed JSON, duplicate/ambiguous owners, missing
paths, replay exemptions without reasons, source-map conflicts, and unwired
gates. Rollback removes the canonical index, validator, paired gate, and
navigation/documentation links; existing runtime owners, source facts, and
historical contract gates remain unchanged.

## Unexecuted items

Only direct Git Bash execution is unexecuted for the environment reason above.
It must be rerun in a functioning Git Bash/CI environment before treating the
shell entry point as directly exercised; the static parity test remains
blocking in the normal Go quality path.

## Final review

The complete tracked and untracked proposal diff was reviewed against the
specification. Changes are limited to the static owner/evidence index,
`tool/contributioncheck` validation, paired repository gates, documentation,
and OpenSpec artifacts. No runtime implementation, port shape, execution
authority, authorization, recovery, effect, or diagnostic-writer semantics
changed. Before archive, `openspec validate
role-scoped-runtime-ports-and-owner-index --strict`, `openspec validate --all`
(152 passed, 0 failed), the owner index gate, documentation impact gate,
roadmap status gate, docs consistency check, and `git diff --check` passed.
