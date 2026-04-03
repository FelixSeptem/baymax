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
        throw "[react-plan-notebook-gate] unable to prepare writable cache directory for $EnvName at $FallbackPath"
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

function Invoke-ReactPlanStep {
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][scriptblock]$Command
    )
    Write-Host "[react-plan-notebook-gate] $Label"
    [void](Invoke-NativeStrict -Label $Label -Command $Command)
}

function Get-ChangedFiles {
    git rev-parse --verify origin/main *> $null
    if ($LASTEXITCODE -eq 0) {
        $mergeBase = (git merge-base HEAD origin/main 2>$null | Select-Object -First 1).Trim()
        if (-not [string]::IsNullOrWhiteSpace($mergeBase)) {
            return @(git diff --name-only --diff-filter=ACMRTUXB "$mergeBase..HEAD" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
        }
    }
    git rev-parse --verify HEAD~1 *> $null
    if ($LASTEXITCODE -eq 0) {
        return @(git diff --name-only --diff-filter=ACMRTUXB HEAD~1..HEAD | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    }
    return @()
}

function Test-ChangedPrefix {
    param(
        [Parameter(Mandatory = $true)][string]$Prefix,
        [Parameter(Mandatory = $true)][string[]]$Files
    )
    foreach ($item in $Files) {
        if ($item.StartsWith($Prefix, [System.StringComparison]::OrdinalIgnoreCase)) {
            return $true
        }
    }
    return $false
}

Invoke-ReactPlanStep -Label "a67 runner lifecycle + hook + boundary suites" -Command {
    go test ./core/runner -run 'Test(ReactPlan|ReactPlanNotebookDoesNotBypass)' -count=1
}

Invoke-ReactPlanStep -Label "a67 config + diagnostics + recorder additive suites" -Command {
    go test ./runtime/config ./runtime/diagnostics ./observability/event -run 'Test(RuntimeReactPlanNotebook|ManagerRuntimeReactPlanNotebook|StoreRunA67|RuntimeRecorderParsesA67|RuntimeRecorderA67)' -count=1
}

Invoke-ReactPlanStep -Label "a67 replay fixture + drift taxonomy suites" -Command {
    go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|ReplayContractPrimaryReasonArbitrationFixtureDriftClassification|PrimaryReasonArbitrationReplayContractFixtureSuite)' -count=1
}

$changedFiles = @(Get-ChangedFiles)
$runnerImpacted = $false
$securityImpacted = $false
$replayImpacted = $false
if ($changedFiles.Count -eq 0) {
    $runnerImpacted = $true
    $securityImpacted = $true
    $replayImpacted = $true
}
else {
    if ((Test-ChangedPrefix -Prefix "core/runner/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "core/types/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "runtime/config/" -Files $changedFiles)) {
        $runnerImpacted = $true
    }
    if ((Test-ChangedPrefix -Prefix "core/runner/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "runtime/config/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "runtime/diagnostics/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "observability/event/" -Files $changedFiles)) {
        $securityImpacted = $true
    }
    if ((Test-ChangedPrefix -Prefix "tool/diagnosticsreplay/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "runtime/diagnostics/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "observability/event/" -Files $changedFiles)) {
        $replayImpacted = $true
    }
}

if ($runnerImpacted) {
    Invoke-ReactPlanStep -Label "impacted-contract suites (runner scope): react contract baseline" -Command {
        pwsh -File scripts/check-react-contract.ps1
    }
}

if ($securityImpacted) {
    Invoke-ReactPlanStep -Label "impacted-contract suites (boundary scope): policy precedence gate" -Command {
        pwsh -File scripts/check-policy-precedence-contract.ps1
    }
    Invoke-ReactPlanStep -Label "impacted-contract suites (boundary scope): sandbox egress + allowlist gate" -Command {
        pwsh -File scripts/check-sandbox-egress-allowlist-contract.ps1
    }
}

if ($replayImpacted) {
    Invoke-ReactPlanStep -Label "impacted-contract suites (replay scope): diagnostics replay contract gate" -Command {
        pwsh -File scripts/check-diagnostics-replay-contract.ps1
    }
}

Invoke-ReactPlanStep -Label "contributioncheck parity suites for react-plan-notebook gate" -Command {
    go test ./tool/contributioncheck -run 'Test(ReactPlanNotebookGateScriptParity|QualityGateIncludesReactPlanNotebookGate|CIIncludesReactPlanNotebookRequiredCheckCandidate)' -count=1
}

Write-Host "[react-plan-notebook-gate] done"
