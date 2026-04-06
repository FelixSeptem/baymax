## A63 Naming Inventory Baseline (2026-04-05)

This inventory captures the initial A63 baseline for legacy naming spread in
active (non-`openspec/**`) repository surface.

### Scan Scope

- Governed roots:
  - `a2a/`, `adapter/`, `cmd/`, `context/`, `core/`, `docs/`, `examples/`
  - `integration/`, `mcp/`, `memory/`, `model/`, `observability/`, `orchestration/`
  - `runtime/`, `scripts/`, `skill/`, `tool/`
  - root docs: `README.md`, `AGENTS.md`, `CONTRIBUTING.md`, `CHANGELOG.md`
- Excluded from this baseline:
  - `openspec/**` (legacy numbering is allowed by policy for historical traceability)

### Commands Used

```powershell
$roots=@('a2a','adapter','cmd','context','core','docs','examples','integration','mcp','memory','model','observability','orchestration','runtime','scripts','skill','tool','AGENTS.md','CHANGELOG.md','CONTRIBUTING.md','README.md')
rg -n --ignore-case 'ca1|ca2|ca3|ca4' $roots
rg -n 'A[0-9]{2,3}' $roots
```

```powershell
# path/file-name baseline
# (collected by enumerating governed files and matching path tokens)
```

### Summary (Initial Snapshot)

- `ca1|ca2|ca3|ca4` content hits: `1887`
- `A[0-9]{2,3}` content hits (non-`openspec/**`): `4880`
- Governed file count: `4540`
- Path/file-name hits:
  - `ca[1-4]` (case-insensitive): `6`
  - uppercase `A[0-9]{2,3}`: `0`
  - lowercase `a[0-9]{2,3}` migration debt: `4050`

### Top Roots by Legacy Hit Count

`ca1|ca2|ca3|ca4` content hits:

- `runtime`: 874
- `context`: 584
- `docs`: 123
- `observability`: 87
- `core`: 84
- `integration`: 76
- `scripts`: 56
- `README.md`: 2
- `cmd`: 1

`A[0-9]{2,3}` content hits:

- `examples`: 3247
- `docs`: 678
- `runtime`: 201
- `tool`: 195
- `integration`: 190
- `scripts`: 150
- `observability`: 114
- `orchestration`: 45
- `core`: 37
- `README.md`: 13

### Top Files by Legacy Hit Count

Top `ca*` files:

- `runtime/config/config.go`: 421
- `context/assembler/assembler_test.go`: 299
- `runtime/config/config_test.go`: 267
- `context/assembler/context_pressure_recovery.go`: 103
- `context/assembler/assembler.go`: 69
- `integration/benchmark_test.go`: 65
- `runtime/diagnostics/store.go`: 61
- `observability/event/runtime_recorder_test.go`: 59
- `runtime/diagnostics/store_test.go`: 59
- `core/runner/runner_test.go`: 51

Top `Axx` files:

- `docs/development-roadmap.md`: 284
- `docs/mainline-contract-test-index.md`: 214
- `docs/runtime-config-diagnostics.md`: 134
- `observability/event/runtime_recorder_test.go`: 79
- `tool/contributioncheck/contract_index_test.go`: 73
- `runtime/diagnostics/store_test.go`: 58
- `runtime/config/readiness_test.go`: 48
- `tool/diagnosticsreplay/arbitration.go`: 36
- `integration/unified_snapshot_contract_test.go`: 32
- `observability/event/runtime_recorder.go`: 32

### Path/File-name Samples

`ca*` path samples:

- `cmd/ca3-threshold-tuning/main.go`
- `context/assembler/context_pressure_recovery.go`
- `docs/ca2-external-retriever-evolution.md`
- `scripts/ca4-benchmark-baseline.env`
- `scripts/check-ca4-benchmark-regression.ps1`
- `scripts/check-ca4-benchmark-regression.sh`

`a[0-9]{2,3}` path samples (migration debt):

- `examples/adapters/_a23-offline-work/...`
- `tool/diagnosticsreplay/testdata/a67_ctx_*.json`
- `tool/diagnosticsreplay/testdata/a68_realtime_*.json`
- `integration/testdata/diagnostics-replay/a47/v1/*.json`
- `integration/testdata/diagnostics-replay/a50/v1/*.json`

### A63 Baseline Artifacts Introduced

- Governed-path matrix (single policy source):
  - `openspec/governance/semantic-labeling-governed-path-matrix.yaml`
- Semantic/legacy mapping (single mapping source):
  - `openspec/governance/semantic-labeling-legacy-mapping.yaml`

These two files are the baseline for subsequent A63 replacement and gate wiring
tasks.
