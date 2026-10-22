# Verification Record

日期：2026-10-22。分支：`propose/bound-durable-store-deployment-scope`；基准：`master` / `45efbb6`。

## Required verification

| Command | Result |
| --- | --- |
| `go test ./...` | exit 0; all packages passed in the full quality gate |
| `go test -race ./...` | exit 0; all packages passed in the full quality gate |
| `golangci-lint run --config .golangci.yml` | exit 0; `0 issues.` |
| `pwsh -File scripts/check-docs-consistency.ps1` | exit 0; documentation consistency passed |
| `openspec validate bound-durable-store-deployment-scope --strict` | exit 0; change is valid |
| `pwsh -File scripts/check-durable-store-deployment-scope-contract.ps1` | exit 0; deployment-scope contract passed |
| `pwsh -File scripts/check-quality-gate.ps1` | exit 1 only at strict `govulncheck`; all 86 preceding steps, including tests, race, lint, contracts, replay, A64 semantic/performance, and full-chain smoke, passed |

## Strict vulnerability advisory

The strict quality gate reached `govulncheck ./...` and stopped with exit 3 because the current Go/toolchain and dependency baseline contain externally reported vulnerabilities. The observed advisory includes `GO-2026-6809` affecting `crypto/x509` in Go `1.26.9`, with the fixed release reported as Go `1.26.10`; the scan also reported additional standard-library and dependency findings. These findings are outside this change's deployment-scope implementation and were not suppressed, downgraded, or used to weaken any contract assertion.

The follow-up direct `govulncheck ./...` invocation could not refresh the vulnerability database in the restricted environment: `vuln.go.dev` was unreachable through the configured proxy. This is recorded as an environment limitation, not as evidence that the advisory is resolved. Strict quality-gate status therefore remains non-zero for the external security baseline, while all change-specific verification remains passing.

The quality-gate PowerShell helper also required a narrow robustness fix: stderr warnings from `go env CGO_ENABLED` are represented as PowerShell `ErrorRecord` values, so the gate now filters those records before normalizing the actual CGO value. This preserves strict warning detection and does not change any runtime or contract threshold.

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`docs/guides/integration.md`, `docs/runtime-module-boundaries.md`, `docs/runtime-config-diagnostics.md`, `docs/mainline-contract-test-index.md`, `docs/development-roadmap.md`, package READMEs
- 维护 owner：core/types, core/checkpoint, core/history, orchestration source owners, planbridge
- 验证证据：上述 Go tests/race/lint, deployment-scope contract gate, OpenSpec validation, docs consistency, and full quality-gate record
