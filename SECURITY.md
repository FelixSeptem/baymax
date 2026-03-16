# Security Policy

## Reporting a Vulnerability

Please report security vulnerabilities through GitHub Security Advisory (private reporting).

- Go to: Security tab -> Advisories -> Report a vulnerability.
- Do not create a public issue for unpatched vulnerabilities.

## Scope

This process covers vulnerabilities in:

- Runtime packages and adapters in this repository.
- Build/test scripts and CI workflows that affect supply-chain safety.

## Response Timeline

- Initial triage acknowledgement: within 3 business days.
- Severity assessment and owner assignment: within 5 business days.
- Remediation target:
  - Critical: 7 calendar days
  - High: 14 calendar days
  - Medium: 30 calendar days
  - Low: best effort in normal release cycle

## Disclosure Process

1. Receive report through GitHub Security Advisory.
2. Triage and classify severity.
3. Prepare fix and validation.
4. Coordinate disclosure timing with reporter.
5. Publish advisory and release notes/changelog entry.

## Supported Versions

Security fixes are prioritized for the active mainline branch.
Backports are best-effort and evaluated case by case.
