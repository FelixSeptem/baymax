# A69 Baseline Scope and Gate Matrix

## 1. Impacted Surface (a69-S0-T01)

Primary implementation surface for `context compression production hardening`:

- `context/assembler`
  - pressure compaction quality gating, fallback class, swap-back/tiering semantics, cold-store governance actions.
- `context/journal`
  - context spill/journal compatibility path and lifecycle recoverability assertions.
- `runtime/config`
  - A69 governance fields (`runtime.context.jit.*`) with `env > file > default`.
- `runtime/diagnostics`
  - additive A69 diagnostics persistence, parser compatibility, nullable/default behavior.
- `integration`
  - replay fixture contracts and run/stream consistency assertions.
- `scripts`
  - A69 production contract gate, quality-gate impacted mapping, benchmark/replay blocking.
- `docs`
  - runtime config/diagnostics field mapping and mainline contract index.

## 2. Boundary Mapping: A69 vs A64 vs A62 (a69-S0-T02)

Ownership split:

- `A69` (`introduce-context-compression-production-hardening-contract-a69`)
  - semantic governance hardening, deterministic fallback/tiering/recovery, replay taxonomy, gate hard-blocking.
- `A64` (`context pressure performance hardening`)
  - non-semantic performance optimization and benchmark engineering.
- `A62` (`delivery usability agent mode example pack`)
  - example-pack delivery and mode onboarding; `context-governed` completion depends on A69 convergence.

Guardrail:

- A69 MUST NOT create a parallel context semantic model.
- A64 MUST NOT redefine context semantic contracts.
- A62 MUST consume stabilized runtime contracts, not define runtime semantics from examples.

## 3. Required Suites Baseline (a69-S0-T03)

A69 required baseline (blocking on impacted changes):

- A69 contract gate:
  - `scripts/check-context-compression-production-contract.sh`
  - `scripts/check-context-compression-production-contract.ps1`
- impacted contract suites chained by A69 gate:
  - `check-context-jit-organization-contract.*`
  - `check-diagnostics-replay-contract.*`
  - `check-context-production-hardening-benchmark-regression.*`
- parity and docs:
  - shell/PowerShell parity via contributioncheck
  - `pwsh -File scripts/check-docs-consistency.ps1`
- required-check mapping:
  - `.github/workflows/ci.yml::context-compression-production-contract-gate`
  - indexed in `docs/mainline-contract-test-index.md`
