#!/usr/bin/env bash
set -euo pipefail

event_path="${1:-${GITHUB_EVENT_PATH:-}}"

if [[ -z "${event_path}" ]]; then
  echo "[contribution-template] usage: bash scripts/check-contribution-template.sh <event.json>"
  echo "[contribution-template] or set GITHUB_EVENT_PATH"
  exit 2
fi

echo "[contribution-template] validating pull request template completeness"
go run ./cmd/contribution-template-check -event "${event_path}"
