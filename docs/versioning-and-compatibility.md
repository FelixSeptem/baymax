# Versioning and Compatibility

## Versioning Policy

This project uses Semantic Versioning notation (`MAJOR.MINOR.PATCH`) for release identification.

- In the pre-`1.0.0` stage, version increments communicate change scope but do **not** imply strict compatibility guarantees.
- `MAJOR`/`MINOR`/`PATCH` labels are best-effort signals for maintainers and users, not a compatibility contract before `1.0.0`.
- Current governance posture stays in `0.x` and does **not** imply `1.0.0/prod-ready` commitments by default.

Pre-release identifiers (for example `-rc.1`) may be used before a stable release.

## Pre-1.x Compatibility Posture

Before `1.0.0`, this repository does **not** provide compatibility commitments for:

- Public API behavior.
- Runtime config field shapes and accepted enum sets.
- Diagnostics/event contract details.

Maintainers still aim to minimize unnecessary disruption and document meaningful behavior changes through changelog and release notes.

## Change Disclosure

When a change has migration impact, maintainers SHOULD document:

- What changed.
- Why it changed.
- Suggested migration direction (if available).

`CHANGELOG.md` remains the primary disclosure entry.

## Maintenance Window

- Supported line: latest minor only.
- Security and bug fixes are prioritized for the latest minor line; older minors are best-effort.

## Go Version Baseline

- Minimum supported Go version: `1.26`.
- CI baseline runs on the version declared in `go.mod`.

## Provider Compatibility Levels

Current provider adapters:

- `model/openai`
- `model/anthropic`
- `model/gemini`

Compatibility levels:

- `stable`: contract covered by mainline tests and documented behavior guarantees.
- `experimental`: available but behavior surface may evolve quickly.
- `internal`: not for external consumption.

Current baseline classification:

- OpenAI/Anthropic/Gemini adapters: `stable` at repository contract level (`Generate`/`Stream` and error taxonomy mappings).
- Provider-specific extended semantics: may evolve quickly in pre-1.x and are documented incrementally in docs/changelog.

## Documentation Source of Truth

To avoid drift, governance policy is maintained in:

- This file (`docs/versioning-and-compatibility.md`)
- `docs/development-roadmap.md` (pre-1 roadmap scope + proposal admission rules)
- `README.md` (contributor-facing release snapshot)
- `CHANGELOG.md` (release-by-release changes)

## Pre-1 Proposal Admission Baseline

Before `1.0.0`, proposals are expected to stay bounded and reviewable:

- Capability additions are allowed in `0.x` when admission fields and quality-gate requirements are satisfied.
- Mandatory fields: `Why now`, risk, rollback, docs impact, validation commands.
- Objective category: each proposal maps to at least one bounded target category from roadmap governance.
- Long-term platformization topics remain documented directions, not near-term execution scope.
