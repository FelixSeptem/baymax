#!/usr/bin/env bash
set -euo pipefail
if [[ -z "${GOCACHE:-}" ]]; then export GOCACHE="$(pwd)/.gocache"; fi
echo "[external-extension-authoring] running external extension authoring conformance profile external_extension_authoring.v1"
go test ./extension ./integration/extensionauthoring ./tool/diagnosticsreplay -run 'Test(DecodeAuthoring|Authoring|EvaluateAuthoring|ReplayContractExternalExtensionAuthoring)' -count=1
echo "[external-extension-authoring] passed profile=external_extension_authoring.v1"
