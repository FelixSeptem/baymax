## 1. Profile contract and admission

- [x] 1.1 Add bounded profile types and validation at the existing OpenAI configuration/adapter boundary; verify missing, malformed, sensitive, oversized, and unsupported-shape profiles fail fast without provider calls.
- [x] 1.2 Add independent capability declarations for structured input, native tool results, streaming, cache usage, and CountTokens; verify absent capabilities remain unavailable/unknown and no synthetic usage is emitted.
- [x] 1.3 Preserve the official OpenAI default path and existing BaseURL behavior; verify current `model/openai` tests pass unchanged before profile opt-in is exercised.

## 2. Offline conformance fixtures and replay

- [x] 2.1 Add a versioned, privacy-bounded profile/conformance fixture for official Responses behavior and one explicitly declared compatible profile; verify canonical serialization and repeated replay are idempotent.
- [x] 2.2 Validate structured role/part ordering, native tool-result correlation, bounded arguments/results, stream event boundaries, first-semantic-event provider fence, and terminal cache usage; verify positive, negative, and boundary cases.
- [x] 2.3 Prove Run/Stream parity and historical `provider_request_projection.v1` compatibility; verify unavailable cache defaults and unsupported CountTokens remain explicit.

## 3. Gates and documentation

- [x] 3.1 Add shell and PowerShell profile/conformance gates with identical offline inputs and outputs; verify privacy, overflow, undeclared-difference, and no-file-mutation checks.
- [x] 3.2 Update `model/README.md`, `docs/mainline-contract-test-index.md`, `docs/runtime-config-diagnostics.md` if profile fields are documented there, and `docs/development-roadmap.md`; verify docs consistency and no duplicate candidate entry.
- [x] 3.3 Record rollback evidence and explicitly unexecuted live-provider checks; verify rollback preserves official OpenAI behavior and historical fixtures.

## 4. Verification and delivery

- [x] 4.1 Run affected `model/openai`, `model/conformance`, replay, and contribution packages with `-count=1`; verify positive, negative, boundary, and parity coverage.
- [x] 4.2 Run `go test ./...`, `go test -race ./...`, and `golangci-lint run --config .golangci.yml`; record environment failures separately from product failures.
- [x] 4.3 Run quality, docs, OpenSpec, Example Impact, profile/conformance gates, and `git diff --check`; only then mark implementation tasks complete and archive through the prescribed script.
