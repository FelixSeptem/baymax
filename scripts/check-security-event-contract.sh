#!/usr/bin/env bash
set -euo pipefail

echo "[security-event-gate] runner security event contracts"
go test ./core/runner -run '^TestSecurityEventContract' -count=1

echo "[security-event-gate] runtime config security event contracts"
go test ./runtime/config -run '^TestSecurityEventContract' -count=1
