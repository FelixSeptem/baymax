## 1. Adapter Template Documentation Structure

- [x] 1.1 Create external adapter template index under `docs/` with MCP/Model/Tool sections.
- [x] 1.2 Add MCP adapter template first, including minimal runnable snippet and boundary notes.
- [x] 1.3 Add Model provider adapter template, including minimal runnable snippet and boundary notes.
- [x] 1.4 Add Tool adapter template, including minimal runnable snippet and boundary notes.
- [x] 1.5 Ensure templates explicitly document intended scope as onboarding skeletons (not production framework).

## 2. Migration Mapping and Error Playbook

- [x] 2.1 Build migration mapping document using capability-domain and code-snippet dual structure.
- [x] 2.2 Add mapping entries for MCP adapter integration old/new patterns.
- [x] 2.3 Add mapping entries for Model adapter integration old/new patterns.
- [x] 2.4 Add mapping entries for Tool adapter integration old/new patterns.
- [x] 2.5 Add common mistakes and replacement patterns section for each adapter category.
- [x] 2.6 Add unified compatibility boundary section: `additive + nullable + default + fail-fast`.

## 3. Entry Navigation and Index Alignment

- [x] 3.1 Update `README.md` with adapter template and migration mapping entry links.
- [x] 3.2 Update API reference/index docs to include adapter onboarding navigation.
- [x] 3.3 Update `docs/mainline-contract-test-index.md` and `docs/development-roadmap.md` for A21 traceability.

## 4. Quality Gate and Contribution Check Alignment

- [x] 4.1 Extend docs consistency checks to cover adapter template and migration mapping links.
- [x] 4.2 Extend `tool/contributioncheck` assertions for adapter template and mapping index consistency.
- [x] 4.3 Ensure quality gate failure messages for missing/stale adapter mapping entries are explicit.

## 5. Validation

- [x] 5.1 Run `pwsh -File scripts/check-docs-consistency.ps1`.
- [x] 5.2 Run `go test ./tool/contributioncheck -count=1`.
- [x] 5.3 Run `go test ./...`.
- [x] 5.4 Run `go test -race ./...`.
- [x] 5.5 Run `golangci-lint run --config .golangci.yml`.
- [x] 5.6 Run `openspec validate introduce-external-adapter-template-and-migration-mapping-a21 --strict`.

