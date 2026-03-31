Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

function Test-WritableDirectory {
    param(
        [Parameter(Mandatory = $false)][string]$Path
    )
    if ([string]::IsNullOrWhiteSpace($Path)) {
        return $false
    }
    try {
        if (-not (Test-Path -LiteralPath $Path)) {
            New-Item -ItemType Directory -Path $Path -Force | Out-Null
        }
        $probe = Join-Path $Path ("._write_probe_" + [Guid]::NewGuid().ToString("N"))
        [System.IO.File]::WriteAllText($probe, "ok")
        Remove-Item -LiteralPath $probe -Force -ErrorAction SilentlyContinue
        return $true
    }
    catch {
        return $false
    }
}

function Ensure-WritableCacheEnv {
    param(
        [Parameter(Mandatory = $true)][string]$EnvName,
        [Parameter(Mandatory = $true)][string]$FallbackPath
    )
    $current = [Environment]::GetEnvironmentVariable($EnvName)
    if (Test-WritableDirectory -Path $current) {
        return
    }
    if (-not (Test-WritableDirectory -Path $FallbackPath)) {
        throw "[observability-export-bundle-gate] unable to prepare writable cache directory for $EnvName at $FallbackPath"
    }
    Set-Item -Path ("Env:" + $EnvName) -Value $FallbackPath
}

Ensure-WritableCacheEnv -EnvName "GOCACHE" -FallbackPath (Join-Path $repoRoot ".gocache")

Write-Host "[observability-export-bundle-gate] runtime config + readiness contracts"
Invoke-NativeStrict -Label "go test ./runtime/config -run 'Test(RuntimeObservabilityConfig|ManagerRuntimeObservabilityInvalidReloadRollsBack|ManagerReadinessPreflightObservability|ManagerReadinessPreflightDiagnosticsBundleOutputUnavailableStrictMapping|ObservabilityReadinessFindingsCoverProfileAndPolicyInvalidCodes|ArbitratePrimaryReasonObservabilityPolicyInvalidOutranksSinkUnavailable)' -count=1" -Command {
    go test ./runtime/config -run 'Test(RuntimeObservabilityConfig|ManagerRuntimeObservabilityInvalidReloadRollsBack|ManagerReadinessPreflightObservability|ManagerReadinessPreflightDiagnosticsBundleOutputUnavailableStrictMapping|ObservabilityReadinessFindingsCoverProfileAndPolicyInvalidCodes|ArbitratePrimaryReasonObservabilityPolicyInvalidOutranksSinkUnavailable)' -count=1
}

Write-Host "[observability-export-bundle-gate] bundle generator + recorder + run/stream contracts"
Invoke-NativeStrict -Label "go test ./runtime/config ./runtime/diagnostics ./observability/event ./integration -run 'Test(ManagerGenerateDiagnosticsBundle|StoreRunObservabilityAdditiveFieldsPersistAndReplayIdempotent|StoreRunObservabilityAdditiveFieldsBoundedCardinality|RuntimeRecorderAutoGeneratesA55DiagnosticsBundleSuccess|RuntimeRecorderAutoGeneratesA55DiagnosticsBundleFailureReason|ObservabilityExportBundleContractRunStreamSemanticEquivalenceSuccess|ObservabilityExportBundleContractRunStreamBundleFailureTaxonomyEquivalent)' -count=1" -Command {
    go test ./runtime/config ./runtime/diagnostics ./observability/event ./integration -run 'Test(ManagerGenerateDiagnosticsBundle|StoreRunObservabilityAdditiveFieldsPersistAndReplayIdempotent|StoreRunObservabilityAdditiveFieldsBoundedCardinality|RuntimeRecorderAutoGeneratesA55DiagnosticsBundleSuccess|RuntimeRecorderAutoGeneratesA55DiagnosticsBundleFailureReason|ObservabilityExportBundleContractRunStreamSemanticEquivalenceSuccess|ObservabilityExportBundleContractRunStreamBundleFailureTaxonomyEquivalent)' -count=1
}

Write-Host "[observability-export-bundle-gate] diagnostics replay observability.v1 contracts"
Invoke-NativeStrict -Label "go test ./tool/diagnosticsreplay -run 'TestReplayContract(PrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|PrimaryReasonArbitrationFixtureDriftClassification|ArbitrationMixedA48A52MemoryCompatibility)' -count=1" -Command {
    go test ./tool/diagnosticsreplay -run 'TestReplayContract(PrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|PrimaryReasonArbitrationFixtureDriftClassification|ArbitrationMixedA48A52MemoryCompatibility)' -count=1
}
