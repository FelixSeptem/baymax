#!/usr/bin/env bash
set -euo pipefail

echo "[security-policy-gate] runner security policy contracts"
go test ./core/runner -run '^TestSecurityPolicyContract' -count=1

echo "[security-policy-gate] runtime config reload rollback contracts"
go test ./runtime/config -run '^TestSecurityPolicyContract' -count=1
