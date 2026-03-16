# Contributing Guide

## Development Setup

Prerequisites:

- Go 1.26+
- golangci-lint
- govulncheck (recommended; required by quality gate)

Install dependencies:

```bash
go mod tidy
```

## Run Quality Checks

Linux/macOS:

```bash
bash scripts/check-quality-gate.sh
```

Windows PowerShell:

```powershell
pwsh -File scripts/check-quality-gate.ps1
```

Minimum local checks before opening PR:

```bash
go test ./...
go test -race ./...
golangci-lint run --config .golangci.yml
```

## Pull Request Process

1. Keep changes focused and scoped.
2. Update docs when behavior/config/contracts change.
3. Add or update tests for behavior changes.
4. Call out compatibility impact and any breaking changes.
5. Ensure CI is green before requesting review.

## Review Checklist (Required)

- Tests updated or justification provided.
- Docs updated for user-visible changes.
- Compatibility impact assessed.
- Breaking changes explicitly marked in PR and changelog.

## Community Conduct

This project follows the code of conduct in `CODE_OF_CONDUCT.md`.
By participating, you agree to uphold that policy.

## Security

For vulnerability reports, use the private process in `SECURITY.md`.
