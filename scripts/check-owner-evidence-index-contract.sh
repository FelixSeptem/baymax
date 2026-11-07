#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"
echo "[owner-evidence-index] canonical index and negative validation"
go test ./tool/contributioncheck -run '^TestValidateOwnerEvidenceIndex$' -count=1
echo "[owner-evidence-index-contract] passed"
