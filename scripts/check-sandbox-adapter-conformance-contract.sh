#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${GOCACHE:-}" ]]; then
  export GOCACHE="$(pwd)/.gocache"
fi

echo "[sandbox-adapter-gate] sandbox manifest profile-pack contracts"
go test ./adapter/manifest -run 'Test(ParseSandboxManifest|ActivateSandboxManifest|SandboxProfilePack)' -count=1

echo "[sandbox-adapter-gate] external adapter conformance backend/session/capability matrix"
go test ./integration/adapterconformance -run 'TestSandboxAdapterConformance' -count=1

echo "[sandbox-adapter-gate] runtime readiness sandbox adapter findings"
go test ./runtime/config -run 'TestManagerReadinessPreflightSandboxAdapter' -count=1

echo "[sandbox-adapter-gate] adapter contract replay sandbox.v1 + mixed tracks"
go test ./integration/adaptercontractreplay -run 'TestReplayContract(SandboxProfilePackTrack|MixedTracksBackwardCompatible|ProfileVersionValidation)' -count=1

