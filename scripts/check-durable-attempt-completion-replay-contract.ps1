Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
Write-Host "[durable-attempt-completion-replay-gate] fixture and offline replay"
if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) { $env:GOCACHE = Join-Path $repoRoot ".gocache-durable-replay" }
New-Item -ItemType Directory -Force -Path $env:GOCACHE | Out-Null
Invoke-NativeStrict -Label "durable/completion replay tests" -Command {
    go test ./tool/diagnosticsreplay -run 'Test(Parse|Evaluate)(DurableAttemptWorkspaceBinding|CompletionSafePointOwnership)' -count=1
}
$forbidden = rg -n 'os/exec|os/Chdir|git |workspace mutation' tool/diagnosticsreplay/durable_attempt_workspace_binding.go tool/diagnosticsreplay/completion_safe_point_ownership.go
if ($LASTEXITCODE -eq 0 -and $forbidden) { throw "replay implementation contains forbidden side effects" }
