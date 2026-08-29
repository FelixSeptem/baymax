#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${REPO_ROOT}"

echo "[agent-runtime-protocol-contract-gate] replay suites"
go test ./core/types ./core/runner ./tool/diagnosticsreplay ./integration ./observability/event ./observability/trace -run 'Test(AgentRuntimeProtocol|EvaluateProtocolFixture|ProtocolDescriptor|ProjectProtocol|ConcurrentRunAdmission|RuntimeRecorderParses|ProtocolDecisionAttributes|EventStreamBinding|RealtimeEventStreamBinding)' -count=1

echo "[agent-runtime-protocol-contract-gate] capability context action admission Run/Stream parity"
if ! rg -n 'ProtocolDescriptor|ProtocolSessionContext|ProtocolRunAdmission|TestRunAndStreamProtocolProjectionSemanticEquivalence' core/types orchestration a2a integration; then
  echo "[agent-runtime-protocol-contract-gate][projection_surface] required capability/context/action/admission/Run/Stream assertions missing" >&2
  exit 1
fi

echo "[agent-runtime-protocol-contract-gate] stream_binding_control_plane_absent"
if rg -n --glob '!openspec/changes/archive/**' 'stream[_-]?binding.*(listener|connection[_-]?manager|event[_-]?store|global[_-]?queue|hosted[_-]?service|gateway)' .; then
  echo "[agent-runtime-protocol-contract-gate][stream_binding_control_plane_absent] hosted binding dependency detected" >&2
  exit 1
fi

echo "[agent-runtime-protocol-contract-gate] control_plane_absent"
if rg -n --glob '!openspec/changes/archive/**' --glob '!openspec/changes/introduce-agent-runtime-protocol-contract/**' --glob '!openspec/specs/agent-runtime-protocol-contract/**' --glob '!scripts/check-agent-runtime-protocol-contract.*' 'agent_runtime_protocol.*(hosted|control[_-]?plane|gateway|session[_-]?service|artifact[_-]?store)' .; then
  echo "[agent-runtime-protocol-contract-gate][control_plane_absent] hosted protocol dependency detected" >&2
  exit 1
fi

echo "[agent-runtime-protocol-contract-gate] passed"
