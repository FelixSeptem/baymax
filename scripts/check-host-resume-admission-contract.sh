#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
echo '[host-resume-admission-contract] native admission, concurrency, Host, replay and example'
go test ./core/types ./core/runner ./host ./integration ./tool/diagnosticsreplay ./tool/contributioncheck ./examples/host-resume-admission -run 'TestHostResume|TestDynamicAction|TestConnectionAdmitsDynamicAction|TestConnectionRejectsDynamicAction' -count=1 -timeout 2m
set +e
invalid_output="$(BAYMAX_HOST_RESUME_ADMISSION_FIXTURE=testdata/host_resume_admission.invalid.json go test ./tool/diagnosticsreplay -run TestHostResumeReplayGoldenAndDrift -count=1 2>&1)"
invalid_exit=$?
set -e
if [[ $invalid_exit -eq 0 || "$invalid_output" != *host_resume.replay_drift* ]]; then
  echo '[host-resume-admission-contract] invalid fixture lacks host_resume.replay_drift failure' >&2
  exit 1
fi
echo "[host-resume-admission-contract] expected drift exit=$invalid_exit classification=host_resume.replay_drift"
smoke="$(go run ./examples/host-resume-admission)"
for marker in authority_denied resume_accepted completed revision_drift duplicate_no_effect resume_in_progress effect_cas_rejected run_stream_parity; do
  if ! grep -Fxq "$marker" <<<"$smoke"; then
    echo "[host-resume-admission-contract] missing marker $marker" >&2
    exit 1
  fi
done
echo '[host-resume-admission-contract] passed'
