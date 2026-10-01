## 1. Contract and normalization

- [x] 1.1 Define the bounded `model_route_intent_admission.v1` input/output schema, verdict enum, reason taxonomy, privacy rules, and canonical digest; verify schema tests cover valid, missing, contradictory, overflow, and privacy-invalid inputs.
- [x] 1.2 Implement provider-neutral normalization of route intent and referenced catalog/admission facts; verify equivalent ordering produces identical canonical identity and digest without I/O or runtime mutation.
- [x] 1.3 Implement deterministic comparison for `satisfied`, `route-gap-confirmed`, and `insufficient-evidence`; verify no caller-order selection, discovery, provider call, credential probe, or second admission state machine is introduced.

## 2. Admission and parity evidence

- [x] 2.1 Reuse existing capability, credential, fallback, readiness, resolver, and catalog-generation facts; verify required/optional capability, credential status, fallback, and generation mismatch cases preserve existing reason semantics.
- [x] 2.2 Add Run/Stream comparison coverage; verify equivalent inputs produce equivalent verdict, selected/fallback identity, generation, and ordered reasons, while later catalog generations do not rewrite recorded facts.
- [x] 2.3 Add positive, negative, boundary, and privacy unit tests; verify no endpoint, credential material, raw provider payload, prompt, reasoning, or unbounded body can enter normalized evidence.

## 3. Replay and gates

- [x] 3.1 Add versioned success, route-gap, insufficient-evidence, overflow, generation-mismatch, privacy, and Run/Stream parity fixtures; verify fixture digests and bounded fields are self-consistent.
- [x] 3.2 Extend diagnostics replay with route-intent evidence parsing and drift classification; verify repeated replay is idempotent, historical optional fields use documented defaults, and unknown fields are ignored safely.
- [x] 3.3 Add contribution boundary and shell/PowerShell gate checks; verify the change introduces no provider SDK dependency, network/discovery path, credential store/probe, global mutable router, raw payload persistence, or parallel readiness/terminal owner.

## 4. Documentation and integrated verification

- [x] 4.1 Update model/runtime documentation and `docs/mainline-contract-test-index.md`; verify the reference-only boundary, verdict semantics, privacy limits, rollback point, and Example Impact Assessment are documented consistently.
- [x] 4.2 Run `openspec validate --all`, `openspec validate establish-model-route-intent-admission-evidence --strict`, and `git diff --check`; verify no roadmap-status drift or missing/invalid Example Impact Assessment is reported. Evidence: strict change validation and all-change validation passed; roadmap-status, Example Impact Assessment, docs-consistency, and diff-check gates passed.
- [x] 4.3 Run focused Go tests and replay/gate suites, then the repository quality commands (`go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`, `scripts/check-quality-gate.*`, `scripts/check-docs-consistency.*`); record exact results and any platform-limited command as an explicit non-pass. Evidence: focused contract/replay/boundary tests, full `go test ./... -count=1 -timeout 15m`, full `go test -race ./... -timeout 20m`, and `golangci-lint run --config .golangci.yml` passed; docs and contract gates passed. The first complete quality-gate run reported an unrelated, non-deterministic A64 diagnostics benchmark degradation of 15.9365% against a 12.0000% threshold; the standalone diagnostics performance gate was rerun and passed at 4.0413%, so the initial aggregate benchmark result remains recorded as a non-stable external failure rather than being represented as a clean first-pass.
- [x] 4.4 Final scope review and rollback rehearsal; verify no runtime model selection changes, no examples/agent-modes behavior change, and removal of the evidence/replay/gate files restores the pre-change behavior. Evidence: diff and dependency scans found no runtime selector wiring, examples/agent-modes edits, new configuration keys, provider SDK/network/discovery/credential-store paths, RuntimeRecorder schema changes, or proposal-number identifiers; rollback is limited to removing the evidence, replay, fixture, gate, documentation, and OpenSpec files, with no persistent migration.
