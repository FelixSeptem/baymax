#!/usr/bin/env bash
set -euo pipefail

export GOCACHE="${PWD}/.gocache-scenario-contract"

echo "[scenario-simulation] offline contract tests"
go test ./integration/scenariosimulation -count=1

if rg -n '"github.com/FelixSeptem/baymax/(runtime|model|context)/' integration/scenariosimulation; then
  echo "[scenario-simulation] forbidden production dependency detected" >&2
  exit 1
fi

if rg -n 'github.com/FelixSeptem/baymax/integration/scenariosimulation' --glob '*.go' -g '!integration/scenariosimulation/**' runtime context model core tool; then
  echo "[scenario-simulation] production package imports test-support detected" >&2
  exit 1
fi

echo "[scenario-simulation] contract gate passed"
