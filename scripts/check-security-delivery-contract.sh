#!/usr/bin/env bash
set -euo pipefail

echo "[security-delivery-gate] runner security delivery contracts"
go test ./core/runner -run '^TestSecurityDeliveryContract' -count=1

echo "[security-delivery-gate] runtime config security delivery contracts"
go test ./runtime/config -run '^TestSecurityDeliveryContract' -count=1
