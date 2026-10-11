#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
echo '[native-execution-event-contract] projection, observer, integration and offline replay'
go test ./core/types ./host/eventadapter ./integration ./tool/diagnosticsreplay ./tool/contributioncheck -run 'TestNativeExecution|TestObserver' -count=1
pattern='"(net/|database/|os"|github.com/FelixSeptem/baymax/(core/runner|runtime/|observability/|orchestration/|model/|mcp/))|^[\t ]*go[\t ]+|make\([\t ]*(chan|map\[)|\.(Run|Stream|Resume|Cancel|Retry|Record|AppendEvent)\('
for file in core/types/native_execution_event.go host/eventadapter/observer.go tool/diagnosticsreplay/native_execution_event.go; do
  effective="$pattern"
  if [[ "$file" == tool/* ]]; then
    effective='"(net/|database/|os"|github.com/FelixSeptem/baymax/(core/runner|runtime/|observability/|orchestration/|model/|mcp/))|^[\t ]*go[\t ]+|make\([\t ]*chan|\.(Run|Stream|Resume|Cancel|Retry|Record|AppendEvent)\('
  fi
  status=0
  rg -n -- "$effective" "$file" && status=0 || status=$?
  if [[ "$status" != 1 ]]; then
    echo '[native-execution-event-contract][ownership] forbidden dependency/worker/cache or scan failure' >&2
    exit 1
  fi
done
smoke="$(go run ./examples/native-execution-events)"
for marker in 'native-events run=completed' 'native-events stream=completed' 'observer=healthy' 'pause-is-terminal=false'; do
  if ! grep -Fxq "$marker" <<< "${smoke//$'\r'/}"; then
    echo "[native-execution-event-contract] missing marker $marker" >&2
    exit 1
  fi
done
echo '[native-execution-event-contract] passed'
