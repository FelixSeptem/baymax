## 1. Governance Policy Realignment

- [x] 1.1 Update `docs/versioning-and-compatibility.md` to declare pre-1.x no-compatibility-commitment policy and latest-minor-only maintenance scope
- [x] 1.2 Update `SECURITY.md` to use security email reporting channel and remove response-time SLA commitments while keeping best-effort workflow
- [x] 1.3 Update `CONTRIBUTING.md` wording to align with Chinese-first templates, required fields, and non-SLA governance posture

## 2. Contribution Template Contract

- [x] 2.1 Refactor `.github/pull_request_template.md` to include required structured sections and mandatory checklist items
- [x] 2.2 Refactor `.github/ISSUE_TEMPLATE/bug_report.md` with required Chinese-first fields for summary, reproduction, expected/actual behavior, and environment metadata
- [x] 2.3 Refactor `.github/ISSUE_TEMPLATE/feature_request.md` with required Chinese-first fields for problem statement, solution, alternatives, and impact

## 3. Blocking Enforcement

- [x] 3.1 Implement or extend repository script to validate PR template completeness and mandatory checklist semantics with deterministic reason codes
- [x] 3.2 Add CI step/job to run contribution-template validation and fail fast on violations
- [x] 3.3 Ensure contribution-template validation is documented as required status check for default merge flow

## 4. Verification and Documentation Consistency

- [x] 4.1 Update README and/or governance entry docs to point to revised policy sources (`versioning`, `SECURITY`, `CONTRIBUTING`)
- [x] 4.2 Add or update tests for template validation logic, including pass/fail cases and stable diagnostics
- [x] 4.3 Run quality baseline checks (`go test ./...`, `go test -race ./...`, `golangci-lint run --config .golangci.yml`) and record results in PR
