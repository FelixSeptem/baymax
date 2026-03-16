# Versioning and Compatibility

## Versioning Policy

This project follows Semantic Versioning (`MAJOR.MINOR.PATCH`).

- `MAJOR`: incompatible public behavior changes.
- `MINOR`: backward-compatible feature additions.
- `PATCH`: backward-compatible bug fixes and non-functional improvements.

Pre-release identifiers (for example `-rc.1`) may be used before a stable release.

## Breaking Change Policy

A change is considered breaking when at least one of the following is true:

- Public API behavior changes in a way existing integrations cannot keep working without modification.
- Runtime config fields or accepted enum values are removed or redefined incompatibly.
- Diagnostics or event contracts remove required fields used by downstream systems.

Every breaking change MUST be documented in:

- `CHANGELOG.md` under a `Breaking Changes` section.
- Release notes with migration guidance.

## Go Version Support Window

- Minimum supported Go version: `1.26`.
- CI baseline runs on the version in `go.mod`.
- New releases SHOULD keep at least one active Go major line support window unless explicitly documented otherwise.

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
- Provider-specific extended semantics: may evolve and are documented incrementally in docs and changelog.

## Documentation Source of Truth

To avoid drift, compatibility commitments are maintained in:

- This file (`docs/versioning-and-compatibility.md`)
- `README.md` (entry links only)
- `CHANGELOG.md` (release-by-release changes)
