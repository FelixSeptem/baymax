Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot
if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}
if ($env:GODEBUG) {
    if ($env:GODEBUG -notmatch "(^|,)goindex=") {
        $env:GODEBUG = "$($env:GODEBUG),goindex=0"
    }
}
else {
    $env:GODEBUG = "goindex=0"
}

function Assert-ContainsLiteral {
    param(
        [Parameter(Mandatory = $true)][string]$Assertion,
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(Mandatory = $true)][string]$Literal
    )

    $fullPath = Join-Path $repoRoot $FilePath
    if (-not (Test-Path -LiteralPath $fullPath)) {
        throw "[realtime-protocol-contract-gate][$Assertion] missing file: $FilePath"
    }
    $content = Get-Content -LiteralPath $fullPath -Raw
    if (-not $content.Contains($Literal)) {
        throw "[realtime-protocol-contract-gate][$Assertion] missing marker '$Literal' in $FilePath"
    }
}

function Assert-PatternAbsentAcrossRepo {
    param(
        [Parameter(Mandatory = $true)][string]$Assertion,
        [Parameter(Mandatory = $true)][string]$Pattern
    )

    $rgPath = Get-Command rg -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty Source
    if (-not [string]::IsNullOrWhiteSpace($rgPath)) {
        $scanOutput = @(& $rgPath -n --glob '!openspec/changes/archive/**' -- $Pattern $repoRoot.Path 2>&1)
        $scanExit = $LASTEXITCODE
        if ($scanExit -eq 0) {
            $preview = ($scanOutput | Where-Object { $null -ne $_ } | Select-Object -First 10 | ForEach-Object {
                    $_.ToString().Trim()
                }) -join "`n"
            throw "[realtime-protocol-contract-gate][$Assertion] unexpected matches found for /$Pattern/:`n$preview"
        }
        if ($scanExit -eq 1) {
            return
        }
        $details = ($scanOutput | Where-Object { $null -ne $_ } | Select-Object -First 10 | ForEach-Object {
                $_.ToString().Trim()
            }) -join "`n"
        throw "[realtime-protocol-contract-gate][$Assertion] rg scan failed for /$Pattern/ (exit=$scanExit):`n$details"
    }

    $archiveRoot = [Regex]::Escape((Join-Path $repoRoot "openspec\changes\archive"))
    $files = Get-ChildItem -Path $repoRoot -Recurse -File | Where-Object {
        $_.FullName -notmatch $archiveRoot
    }

    $matches = @()
    foreach ($file in $files) {
        if (-not (Test-Path -LiteralPath $file.FullName)) {
            continue
        }
        $hit = $null
        try {
            $hit = Select-String -Path $file.FullName -Pattern $Pattern -ErrorAction Stop
        }
        catch [System.Management.Automation.ItemNotFoundException] {
            continue
        }
        catch [System.IO.FileNotFoundException] {
            continue
        }
        catch {
            continue
        }
        if ($hit) {
            $matches += $hit
            if ($matches.Count -ge 10) {
                break
            }
        }
    }

    if ($matches.Count -gt 0) {
        $preview = ($matches | Select-Object -First 10 | ForEach-Object {
                "$($_.Path):$($_.LineNumber): $($_.Line.Trim())"
            }) -join "`n"
        throw "[realtime-protocol-contract-gate][$Assertion] unexpected matches found for /$Pattern/:`n$preview"
    }
}

function Assert-NoParallelA68Changes {
    param(
        [Parameter(Mandatory = $true)][string]$Assertion
    )

    $changeRoot = Join-Path $repoRoot "openspec/changes"
    $canonical = "introduce-realtime-event-protocol-and-interrupt-resume-contract-a68"
    $violations = @()
    $dirs = Get-ChildItem -Path $changeRoot -Directory | Where-Object { $_.Name -ne "archive" }
    foreach ($dir in $dirs) {
        $lower = $dir.Name.ToLowerInvariant()
        if ($dir.Name -ne $canonical -and
            $lower.Contains("realtime") -and
            ($lower.Contains("interrupt") -or $lower.Contains("resume") -or $lower.Contains("protocol"))) {
            $violations += $dir.Name
        }
    }
    if ($violations.Count -gt 0) {
        throw "[realtime-protocol-contract-gate][$Assertion] parallel realtime proposal detected: $($violations -join ', ')"
    }
}

function Resolve-A68ChangeDir {
    $active = "openspec/changes/introduce-realtime-event-protocol-and-interrupt-resume-contract-a68"
    $activeFull = Join-Path $repoRoot $active
    if (Test-Path -LiteralPath $activeFull -PathType Container) {
        return $active
    }

    $archiveRoot = Join-Path $repoRoot "openspec/changes/archive"
    if (Test-Path -LiteralPath $archiveRoot -PathType Container) {
        $candidate = Get-ChildItem -Path $archiveRoot -Directory |
            Where-Object { $_.Name -like "*introduce-realtime-event-protocol-and-interrupt-resume-contract-a68" } |
            Select-Object -First 1
        if ($candidate) {
            return "openspec/changes/archive/$($candidate.Name)"
        }
    }

    throw "[realtime-protocol-contract-gate] unable to locate A68 change directory in active or archive paths"
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

function Invoke-RealtimeStep {
    param(
        [Parameter(Mandatory = $true)][string]$Label,
        [Parameter(Mandatory = $true)][scriptblock]$Command
    )
    Write-Host "[realtime-protocol-contract-gate] $Label"
    & $Command
}

$a68ChangeDir = Resolve-A68ChangeDir

Invoke-RealtimeStep -Label "assertion realtime_control_plane_absent: design marker" -Command {
    Assert-ContainsLiteral -Assertion "realtime_control_plane_absent" -FilePath "$a68ChangeDir/design.md" -Literal "不引入平台化控制面。"
}

Invoke-RealtimeStep -Label "assertion realtime_control_plane_absent: gate spec marker" -Command {
    Assert-ContainsLiteral -Assertion "realtime_control_plane_absent" -FilePath "$a68ChangeDir/specs/go-quality-gate/spec.md" -Literal "realtime_control_plane_absent"
}

Invoke-RealtimeStep -Label "assertion realtime_control_plane_absent: active change set closure" -Command {
    Assert-NoParallelA68Changes -Assertion "realtime_control_plane_absent"
}

Invoke-RealtimeStep -Label "assertion realtime_control_plane_absent: reject hosted realtime control-plane config drift" -Command {
    Assert-PatternAbsentAcrossRepo -Assertion "realtime_control_plane_absent" -Pattern "runtime\.realtime\.[a-zA-Z0-9_.-]*(control_plane|controlplane|gateway|connection_router|session_router|managed_connection|hosted_realtime|realtime_service)"
}

Invoke-RealtimeStep -Label "assertion a68_same_domain_closure: roadmap marker" -Command {
    Assert-ContainsLiteral -Assertion "a68_same_domain_closure" -FilePath "docs/development-roadmap.md" -Literal "A68 realtime 同域增量需求（事件类型扩展、中断恢复语义、顺序/幂等、回放/门禁）仅允许在 A68 内以增量任务吸收，不再新增平行 realtime 提案。"
}

Write-Host "[realtime-protocol-contract-gate] a68 runtime config governance suites"
Invoke-NativeStrict -Label "go test ./runtime/config -run 'Test(RuntimeRealtimeConfig|ManagerRuntimeRealtime)' -count=1" -Command {
    go test ./runtime/config -run 'Test(RuntimeRealtimeConfig|ManagerRuntimeRealtime)' -count=1
}

Write-Host "[realtime-protocol-contract-gate] a68 realtime envelope + runner parity suites"
Invoke-NativeStrict -Label "go test ./core/types ./core/runner -run 'Test(ParseRealtimeEventEnvelope|RealtimeEventEnvelope|RealtimeRunStream|RealtimeSequenceGapAndOrderClassification)' -count=1" -Command {
    go test ./core/types ./core/runner -run 'Test(ParseRealtimeEventEnvelope|RealtimeEventEnvelope|RealtimeRunStream|RealtimeSequenceGapAndOrderClassification)' -count=1
}

Write-Host "[realtime-protocol-contract-gate] a68 diagnostics recorder additive suites"
Invoke-NativeStrict -Label "go test ./runtime/diagnostics ./observability/event -run 'Test(StoreRunA68|RuntimeRecorderParsesA68RealtimeAdditiveFields|RuntimeRecorderA68ParserCompatibilityAdditiveNullableDefault)' -count=1" -Command {
    go test ./runtime/diagnostics ./observability/event -run 'Test(StoreRunA68|RuntimeRecorderParsesA68RealtimeAdditiveFields|RuntimeRecorderA68ParserCompatibilityAdditiveNullableDefault)' -count=1
}

Write-Host "[realtime-protocol-contract-gate] a68 replay fixture + drift taxonomy suites"
Invoke-NativeStrict -Label "go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|ReplayContractPrimaryReasonArbitrationFixtureDriftClassification|PrimaryReasonArbitrationReplayContractFixtureSuite|PrimaryReasonArbitrationReplayContractDriftGuardFailFast)' -count=1" -Command {
    go test ./tool/diagnosticsreplay ./integration -run 'Test(ReplayContractPrimaryReasonArbitrationFixtureSuccessAndDeterministicOutput|ReplayContractPrimaryReasonArbitrationFixtureDriftClassification|PrimaryReasonArbitrationReplayContractFixtureSuite|PrimaryReasonArbitrationReplayContractDriftGuardFailFast)' -count=1
}

$changedFiles = @(Get-ChangedFiles)
$parityImpacted = $false
$replayImpacted = $false
if ($changedFiles.Count -eq 0) {
    $parityImpacted = $true
    $replayImpacted = $true
}
else {
    if ((Test-ChangedPrefix -Prefix "core/runner/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "core/types/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "runtime/config/" -Files $changedFiles)) {
        $parityImpacted = $true
    }
    if ((Test-ChangedPrefix -Prefix "tool/diagnosticsreplay/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "integration/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "runtime/diagnostics/" -Files $changedFiles) -or
        (Test-ChangedPrefix -Prefix "observability/event/" -Files $changedFiles)) {
        $replayImpacted = $true
    }
}

if ($parityImpacted) {
    Write-Host "[realtime-protocol-contract-gate] impacted-contract suites (runner scope): react contract baseline"
    Invoke-NativeStrict -Label "pwsh -File scripts/check-react-contract.ps1" -Command {
        pwsh -File scripts/check-react-contract.ps1
    }
    Write-Host "[realtime-protocol-contract-gate] impacted-contract suites (runner scope): react plan notebook gate"
    Invoke-NativeStrict -Label "pwsh -File scripts/check-react-plan-notebook-contract.ps1" -Command {
        pwsh -File scripts/check-react-plan-notebook-contract.ps1
    }
}

if ($replayImpacted) {
    Write-Host "[realtime-protocol-contract-gate] impacted-contract suites (replay scope): diagnostics replay contract gate"
    Invoke-NativeStrict -Label "pwsh -File scripts/check-diagnostics-replay-contract.ps1" -Command {
        pwsh -File scripts/check-diagnostics-replay-contract.ps1
    }
}

Write-Host "[realtime-protocol-contract-gate] contributioncheck parity suites for realtime protocol gate"
Invoke-NativeStrict -Label "go test ./tool/contributioncheck -run 'Test(RealtimeProtocolGateScriptParity|QualityGateIncludesRealtimeProtocolGate|CIIncludesRealtimeProtocolRequiredCheckCandidate|RealtimeProtocolRoadmapAndContractIndexClosureMarkers)' -count=1" -Command {
    go test ./tool/contributioncheck -run 'Test(RealtimeProtocolGateScriptParity|QualityGateIncludesRealtimeProtocolGate|CIIncludesRealtimeProtocolRequiredCheckCandidate|RealtimeProtocolRoadmapAndContractIndexClosureMarkers)' -count=1
}

Write-Host "[realtime-protocol-contract-gate] done"
