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
        throw "[policy-precedence-gate] unable to prepare writable cache directory for $EnvName at $FallbackPath"
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

function Invoke-PolicyPrecedenceStep {
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][scriptblock]$Command
    )
    Write-Host "[policy-precedence-gate] $Label"
    [void](Invoke-NativeStrict -Label $Label -Command $Command)
}

Invoke-PolicyPrecedenceStep -Label "runtime policy precedence config/evaluator/rollback suites" -Command {
    go test ./runtime/config -run 'Test(RuntimePolicyConfig|EvaluateRuntimePolicyDecision|ManagerRuntimePolicyInvalidReloadRollsBack|ManagerReadinessPreflightPolicyCandidatesWinnerMetadata|ManagerReadinessAdmissionPolicyDecisionTraceFields)' -count=1
}

Invoke-PolicyPrecedenceStep -Label "runner run/stream parity and deny side-effect-free suites" -Command {
    go test ./core/runner -run 'Test(ActionGateRunAndStreamDenySemanticsEquivalent|ActionGateRunAndStreamTimeoutSemanticsEquivalent|SecurityEventContractSandboxPolicyDenyRunAndStreamEquivalent)' -count=1
}

Invoke-PolicyPrecedenceStep -Label "diagnostics and recorder additive/replay-idempotent suites" -Command {
    go test ./runtime/diagnostics ./observability/event -run 'Test(StoreRunPolicyPrecedenceAdditiveFieldsPersistAndReplayIdempotent|StoreRunPolicyPrecedenceAdditiveFieldsBoundedCardinality|RuntimeRecorderParsesPolicyPrecedenceAdditiveFields|RuntimeRecorderPolicyPrecedenceParserCompatibilityAdditiveNullableDefault)' -count=1
}

Invoke-PolicyPrecedenceStep -Label "policy stack replay and drift taxonomy suites" -Command {
    go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPolicyPrecedenceFixture|ReplayContractMixedPolicyPrecedenceReactSandboxEgressCompatibility|ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|ReplayContractPrimaryReasonArbitrationFixtureDriftClassification|PrimaryReasonArbitrationReplayContractFixtureSuite|PrimaryReasonArbitrationReplayContractDriftGuardFailFast)' -count=1
}

Invoke-PolicyPrecedenceStep -Label "docs parity suites" -Command {
    pwsh -File scripts/check-docs-consistency.ps1
}

Write-Host "[policy-precedence-gate] done"

