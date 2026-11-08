#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

findings_path="docs/upstream/baymax-findings.md"
evidence_path="docs/baymax-contract-evidence.md"
current_pin="v0.0.0-20261103091945-d2e08cd6dc8c"
legacy_pin="v0.0.0-20261022053159-3d0ab5949790"

test -f "${findings_path}" || { echo "[upstream-findings-current-pin] missing ${findings_path}" >&2; exit 1; }
test -f "${evidence_path}" || { echo "[upstream-findings-current-pin] missing ${evidence_path}" >&2; exit 1; }

require_marker() {
  local marker="$1"
  local path="$2"
  rg -Fq -- "${marker}" "${path}" || {
    echo "[upstream-findings-current-pin] missing marker '${marker}' in ${path}" >&2
    exit 1
  }
}

require_marker "${current_pin}" "${findings_path}"
require_marker "${current_pin}" "${evidence_path}"
require_marker "${legacy_pin}" "${findings_path}"
require_marker "${legacy_pin}" "${evidence_path}"
require_marker "historical/legacy" "${findings_path}"
require_marker "historical/legacy" "${evidence_path}"

for marker in \
  "Child/composer continuation" \
  "Parallel/vote team continuation" \
  "Source history and projection recovery" \
  "Mailbox lifecycle" \
  "Native cancellation and timeout" \
  "Native causal retry" \
  "Provider live-smoke"; do
  require_marker "${marker}" "${findings_path}"
done

for marker in \
  '| child / composer |' \
  '| `team_parallel` |' \
  '| `team_vote` |' \
  '| workflow |' \
  '| mailbox lifecycle |' \
  '| native cancellation / timeout |' \
  '| native causal retry |' \
  '| provider live paths |'; do
  require_marker "${marker}" "${evidence_path}"
done

combined="$(cat "${findings_path}" "${evidence_path}")"
for marker in same-host cross-machine source/projection non-atomic recovery-required skipped-unconfigured unsupported; do
  grep -Fq -- "${marker}" <<<"${combined}" || {
    echo "[upstream-findings-current-pin] missing limitation marker '${marker}'" >&2
    exit 1
  }
done

if rg -qi 'all execution forms.{0,80}(support|supported).{0,80}durable replay' "${findings_path}" "${evidence_path}"; then
  echo '[upstream-findings-current-pin] evidence overclaims durable replay for all execution forms' >&2
  exit 1
fi
if rg -qi 'cross-machine coordination (is|remains) (supported|verified)|source/projection (atomic commit|atomic transaction|atomicity is supported)' "${findings_path}" "${evidence_path}"; then
  echo '[upstream-findings-current-pin] evidence overclaims cross-machine or atomic support' >&2
  exit 1
fi
if ! rg -qi 'not (a|an) (Baymax )?(upstream )?bug' "${findings_path}" "${evidence_path}"; then
  echo '[upstream-findings-current-pin] skipped-unconfigured must not be classified as a bug' >&2
  exit 1
fi

echo '[upstream-findings-current-pin] passed'
