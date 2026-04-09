Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$roadmapPath = "docs/development-roadmap.md"
$archiveIndexPath = "openspec/changes/archive/INDEX.md"

function Get-StringSet {
    param(
        [Parameter(Mandatory = $false)][AllowEmptyCollection()][string[]]$Values
    )
    $set = @{}
    if ($null -eq $Values) {
        return $set
    }
    foreach ($value in $Values) {
        $trimmed = ([string]$value).Trim()
        if ([string]::IsNullOrWhiteSpace($trimmed)) {
            continue
        }
        $set[$trimmed] = $true
    }
    return $set
}

function Get-RoadmapStatusSlug {
    param(
        [Parameter(Mandatory = $true)][string]$Line
    )
    if ($Line -match '`([^`]+)`') {
        $slug = $Matches[1].Trim()
        if (-not [string]::IsNullOrWhiteSpace($slug)) {
            return $slug
        }
    }
    if ($Line -match '([a-z0-9]+(?:-[a-z0-9]+)+)') {
        return $Matches[1].Trim()
    }
    return ""
}

if (-not (Test-Path $roadmapPath)) {
    throw "[roadmap-status-drift] missing required roadmap file: $roadmapPath"
}
if (-not (Test-Path $archiveIndexPath)) {
    throw "[roadmap-status-drift] missing required archive index file: $archiveIndexPath"
}

$openspecOutput = Invoke-NativeCaptureStrict -Label "openspec list --json" -Command {
    openspec list --json
}
$openspecText = ($openspecOutput | ForEach-Object {
        if ($null -eq $_) { return "" }
        if ($_ -is [System.Management.Automation.ErrorRecord]) { return $_.ToString() }
        return [string]$_
    }) -join "`n"
$openspecPayload = $openspecText | ConvertFrom-Json
$activeChanges = @($openspecPayload.changes |
    Where-Object { ([string]$_.status).Trim().ToLowerInvariant() -eq "in-progress" } |
    ForEach-Object { ([string]$_.name).Trim() } |
    Where-Object { -not [string]::IsNullOrWhiteSpace($_) } |
    Sort-Object -Unique)

$archivedChanges = New-Object 'System.Collections.Generic.List[string]'
$archiveLines = Get-Content -Path $archiveIndexPath
foreach ($line in $archiveLines) {
    $trimmed = ([string]$line).Trim()
    if ($trimmed -match '^\-\s+[0-9]+\s*->\s*(.+)$') {
        $slug = $Matches[1].Trim()
        if (-not [string]::IsNullOrWhiteSpace($slug)) {
            $archivedChanges.Add($slug) | Out-Null
        }
    }
}
$archivedChanges = @($archivedChanges | Sort-Object -Unique)

$roadmapInProgress = New-Object 'System.Collections.Generic.List[string]'
$roadmapArchived = New-Object 'System.Collections.Generic.List[string]'
$roadmapLines = Get-Content -Path $roadmapPath
$inCurrentStatus = $false
$mode = ""
foreach ($line in $roadmapLines) {
    $trimmed = ([string]$line).Trim()
    if (-not $inCurrentStatus) {
        if ($trimmed -like "## 当前状态*") {
            $inCurrentStatus = $true
        }
        continue
    }
    if ($trimmed.StartsWith("## ")) {
        break
    }

    if ($trimmed -match '^\-\s*进行中：') {
        $mode = "in-progress"
        continue
    }
    if ($trimmed -match '^\-\s*已归档：') {
        $mode = "archived"
        continue
    }
    if ($trimmed -match '^\-\s*候选：') {
        $mode = "candidate"
        continue
    }
    if (-not $trimmed.StartsWith("-")) {
        continue
    }
    if ($mode -ne "in-progress" -and $mode -ne "archived") {
        continue
    }
    $slug = Get-RoadmapStatusSlug -Line $trimmed
    if ([string]::IsNullOrWhiteSpace($slug)) {
        continue
    }
    if ($mode -eq "in-progress") {
        $roadmapInProgress.Add($slug) | Out-Null
        continue
    }
    $roadmapArchived.Add($slug) | Out-Null
}
$roadmapInProgress = @($roadmapInProgress | Sort-Object -Unique)
$roadmapArchived = @($roadmapArchived | Sort-Object -Unique)

$activeSet = Get-StringSet -Values $activeChanges
$archivedSet = Get-StringSet -Values $archivedChanges
$roadmapInProgressSet = Get-StringSet -Values $roadmapInProgress
$roadmapArchivedSet = Get-StringSet -Values $roadmapArchived

$issues = New-Object 'System.Collections.Generic.List[string]'

foreach ($change in $activeChanges) {
    if ($roadmapInProgressSet.ContainsKey($change)) {
        continue
    }
    $issues.Add("[roadmap-status-drift] roadmap missing in-progress change from openspec list: $change") | Out-Null
}

foreach ($change in @($roadmapInProgressSet.Keys | Sort-Object)) {
    if ($activeSet.ContainsKey($change)) {
        continue
    }
    if ($archivedSet.ContainsKey($change)) {
        $issues.Add("[roadmap-status-drift] roadmap marks archived change as in-progress: $change") | Out-Null
        continue
    }
    $issues.Add("[roadmap-status-drift] roadmap in-progress entry is not active in openspec list: $change") | Out-Null
}

foreach ($change in @($roadmapArchivedSet.Keys | Sort-Object)) {
    if ($archivedSet.ContainsKey($change)) {
        continue
    }
    if ($activeSet.ContainsKey($change)) {
        $issues.Add("[roadmap-status-drift] roadmap marks active change as archived: $change") | Out-Null
        continue
    }
    $issues.Add("[roadmap-status-drift] roadmap archived entry is not present in archive index: $change") | Out-Null
}

if ($issues.Count -gt 0) {
    foreach ($issue in $issues) {
        Write-Host $issue
    }
    Write-Host "hint: sync docs/development-roadmap.md current status with openspec list --json and openspec/changes/archive/INDEX.md."
    Write-Host "hint: expected deterministic status authority sources are active=in-progress changes and archive index."
    throw "[roadmap-status-drift] openspec roadmap status consistency failed"
}

Write-Host "[openspec-roadmap-status-consistency] passed"
