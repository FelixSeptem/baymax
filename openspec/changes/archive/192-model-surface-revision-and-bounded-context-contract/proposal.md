## Why

Baymax can validate host-owned context references and compare provider request
facts, but an integrator cannot yet obtain a bounded, deterministic answer to
two connected questions: whether a resolver returned safely bounded material,
and which safe revision summaries of configuration, prompt, and tool schema
reached a particular provider-neutral request projection.  This leaves content
growth and model-visible drift detectable only through ad hoc host code.

## What Changes

- Bound every host resolver material's role, content, and aggregate resolved
  material size.  Resolution rejects the complete descriptor before assembly
  when any bound is exceeded; it never truncates, injects a prefix, or adds a
  runtime configuration key.
- Add an explicit opt-in, provider-neutral model surface summary to request
  projection evidence.  The summary carries only fixed-size digest, length,
  revision, and availability facts for eligible configuration, prompt, and
  tool-schema inputs; missing input remains `unavailable` and is never inferred
  from tool ordering or another projection field.
- Extend deterministic canonicalization, offline replay, drift classification,
  and Run/Stream parity evidence for resolver limits and model-surface facts.
- Document exact bounds, privacy exclusions, opt-in behavior, owner boundaries,
  replay scope, rollback, and the absence of new diagnostics or configuration
  semantics.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `host-context-assembly-handoff`: host-owned resolver material gains explicit
  bounded admission semantics before material can enter context assembly.
- `provider-request-projection-and-cache-observability`: request projection
  gains an opt-in, bounded model-surface summary with deterministic evidence and
  parity/replay behavior.

## Impact

Primary owners are `context/hostprojection` and `model/conformance`; provider
adapters consume the resulting provider-neutral facts without adding a provider
SDK dependency to `context/*`.  The change affects the public library contract,
fixtures, drift gates, and documentation.  It does not create a prompt registry,
persist prompt or configuration bodies, add a RuntimeRecorder writer or
QueryRuns field, change `env > file > default`, or treat model-visible facts as
authorization, terminal, source, or effect evidence.  Rollback removes the
opt-in summary and rejects no new material; callers can omit the summary and
remain on the existing request projection shape.

## Example Impact Assessment

无需示例变更（附理由）：该 change 修改的是 host-resolver admission 与
provider-neutral replay evidence；`examples/agent-modes` 的配置、runtime
path、semantic anchor、expected markers 和 rollback notes 均不变。离线
fixture、replay 和 dedicated gate 提供可执行证据，不引入新的 agent-mode
示例路径。

## Public Semantic Documentation Commitment

- 对外语义变化：是
- 文档同步状态：已同步
- 维护文档：`README.md`、`context/hostprojection/README.md`、`model/README.md`、`docs/runtime-module-boundaries.md`、`docs/runtime-config-diagnostics.md`、`docs/mainline-contract-test-index.md`、`docs/development-roadmap.md`
- 维护 owner：`context/hostprojection`、`model/conformance` 和 documentation maintainers
- 验证证据：`go test ./context/hostprojection ./model/conformance ./tool/diagnosticsreplay`、dedicated shell/PowerShell gate、`pwsh -File scripts/check-openspec-documentation-impact.ps1`、`pwsh -File scripts/check-docs-consistency.ps1`、`openspec validate --all`

## Documentation Impact Assessment

| area | outcome | affected paths | owner | evidence |
| --- | --- | --- | --- | --- |
| architecture | 修改文档 | `docs/runtime-module-boundaries.md` | architecture maintainers | boundary-focused package tests and gate |
| host context API | 修改文档 | `context/hostprojection/README.md` | context maintainers | resolver limit tests |
| model request evidence | 修改文档 | `model/README.md` | model maintainers | request projection and replay tests |
| configuration and diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | assertion of constructor-only/no-config/no-writer semantics |
| contract index | 修改文档 | `docs/mainline-contract-test-index.md` | quality maintainers | fixture and gate references |
| public entry point | 修改文档 | `README.md` | documentation maintainers | library capability summary |
| roadmap | 修改文档 | `docs/development-roadmap.md` | documentation maintainers | roadmap-status consistency check |
| components | 修改文档 | `context/hostprojection/README.md`、`model/README.md` | context/model maintainers | focused package tests |
| configuration | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | no-config assertion and docs consistency gate |
| contract/API | 修改文档 | `README.md`、`docs/runtime-module-boundaries.md` | API/architecture maintainers | OpenSpec validation and focused contract gates |
| diagnostics | 修改文档 | `docs/runtime-config-diagnostics.md` | runtime maintainers | RuntimeRecorder/QueryRuns exclusion assertion |
| examples | 修改文档 | `context/hostprojection/README.md`、`model/README.md` | context/model maintainers | Contract documentation records the behavior; existing examples/agent-modes paths and markers remain unchanged. |
| CLI/integration | 修改文档 | `docs/mainline-contract-test-index.md` | quality maintainers | dual-platform dedicated gates |
| best practices | 修改文档 | `README.md`、`model/README.md` | documentation maintainers | privacy/default behavior tests |
