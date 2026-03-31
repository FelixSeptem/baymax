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
        throw "[react-contract-gate] unable to prepare writable cache directory for $EnvName at $FallbackPath"
    }
    Set-Item -Path ("Env:" + $EnvName) -Value $FallbackPath
}

Ensure-WritableCacheEnv -EnvName "GOCACHE" -FallbackPath (Join-Path $repoRoot ".gocache")

if ($env:GODEBUG) {
    if ($env:GODEBUG -notmatch "(^|,)goindex=") {
        $env:GODEBUG = "$($env:GODEBUG),goindex=0"
    }
}
else {
    $env:GODEBUG = "goindex=0"
}

function Invoke-ReactStep {
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][scriptblock]$Command
    )
    Write-Host "[react-contract-gate] $Label"
    [void](Invoke-NativeStrict -Label $Label -Command $Command)
}

Invoke-ReactStep -Label "runner react taxonomy and budget suites" -Command {
    go test ./core/runner -run 'Test(RunAndStreamToolCallLimitFailFast|ResolveReactTerminationReasonDeterministicMapping|StreamReactDuplicateToolCallEventsAreIdempotent|StreamReactCancellationUsesCanonicalTerminationReason|StreamReactToolDispatchFailureUsesCanonicalTerminationReason)' -count=1
}

Invoke-ReactStep -Label "integration react parity + readiness + sandbox suites" -Command {
    go test ./integration -run 'Test(ReactLoopRunStreamParityIntegrationContract|RuntimeReadinessAdmissionReact|SandboxExecutionIsolationContractReactActionResolutionRunStreamParity|SandboxExecutionIsolationContractReactFallbackTaxonomyAndCountersParity|SandboxExecutionIsolationContractReactCapabilityMismatchRunStreamParity)' -count=1
}

Invoke-ReactStep -Label "runtime readiness react mapping suites" -Command {
    go test ./runtime/config -run 'Test(ManagerReadinessPreflightReact|ArbitratePrimaryReasonReactProviderUnsupportedOutranksRecoverableReactFindings)' -count=1
}

Invoke-ReactStep -Label "provider tool-calling canonicalization suites" -Command {
    go test ./model/openai ./model/anthropic ./model/gemini ./model/providererror ./model/toolcontract -count=1
}

Invoke-ReactStep -Label "diagnostics replay react.v1 suites" -Command {
    go test ./tool/diagnosticsreplay -run 'TestReplayContract(PrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|PrimaryReasonArbitrationFixtureDriftClassification|ArbitrationMixedA48A52MemoryCompatibility)' -count=1
}

Write-Host "[react-contract-gate] done"
