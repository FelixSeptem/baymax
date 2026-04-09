Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[agent-mode-real-logic-contract] validating real runtime entrypoints"

$mainFiles = Get-ChildItem -Path "examples/agent-modes" -Recurse -Filter main.go -File |
Where-Object { $_.FullName -notmatch "\\internal\\" } |
Sort-Object -Property FullName

if ($mainFiles.Count -eq 0) {
    throw "[agent-mode-real-logic-contract] no agent-mode main.go found"
}

$simulatedDependencyViolations = New-Object 'System.Collections.Generic.List[string]'
$placeholderRegressions = New-Object 'System.Collections.Generic.List[string]'
$missingRuntimePath = New-Object 'System.Collections.Generic.List[string]'

foreach ($item in $mainFiles) {
    $raw = Get-Content -Path $item.FullName -Raw

    if ($raw.Contains("examples/agent-modes/internal/agentmode")) {
        $simulatedDependencyViolations.Add($item.FullName) | Out-Null
    }

    $hasRunnerImport = $raw.Contains("github.com/FelixSeptem/baymax/core/runner")
    $hasLocalImport = $raw.Contains("github.com/FelixSeptem/baymax/tool/local")
    $hasRuntimeImport = $raw.Contains("github.com/FelixSeptem/baymax/runtime/config")
    $hasRunnerNew = $raw.Contains("runner.New(")
    if (-not ($hasRunnerImport -and $hasLocalImport -and $hasRuntimeImport -and $hasRunnerNew)) {
        $missingRuntimePath.Add($item.FullName) | Out-Null
    }

    $hasPathMarker = $raw.Contains("verification.mainline_runtime_path=")
    $hasFinalMarker = $raw.Contains("result.final_answer=")
    $hasSignatureMarker = $raw.Contains("result.signature=")
    if (-not ($hasPathMarker -and $hasFinalMarker -and $hasSignatureMarker)) {
        $placeholderRegressions.Add($item.FullName) | Out-Null
    }
}

if ($simulatedDependencyViolations.Count -gt 0) {
    Write-Host "[agent-mode-real-logic-contract][agent-mode-simulated-engine-dependency] prohibited simulation dependency detected:"
    foreach ($path in $simulatedDependencyViolations) {
        Write-Host "  - $path"
    }
}
if ($placeholderRegressions.Count -gt 0) {
    Write-Host "[agent-mode-real-logic-contract][agent-mode-placeholder-output-regression] required real runtime output markers missing:"
    foreach ($path in $placeholderRegressions) {
        Write-Host "  - $path"
    }
}
if ($missingRuntimePath.Count -gt 0) {
    Write-Host "[agent-mode-real-logic-contract][agent-mode-missing-mainline-runtime-path] required runtime wiring missing:"
    foreach ($path in $missingRuntimePath) {
        Write-Host "  - $path"
    }
}

if ($simulatedDependencyViolations.Count -gt 0 -or $placeholderRegressions.Count -gt 0 -or $missingRuntimePath.Count -gt 0) {
    exit 1
}

Write-Host "[agent-mode-real-logic-contract] all agent-mode entrypoints satisfy real runtime contract"
