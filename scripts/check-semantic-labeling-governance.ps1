Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "lib/native-strict.ps1")

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$matrixPath = Join-Path $repoRoot "openspec/governance/semantic-labeling-governed-path-matrix.yaml"
$mappingPath = Join-Path $repoRoot "openspec/governance/semantic-labeling-legacy-mapping.yaml"
$baselinePath = Join-Path $repoRoot "openspec/governance/semantic-labeling-regression-baseline.csv"

foreach ($required in @($matrixPath, $mappingPath, $baselinePath)) {
    if (-not (Test-Path -LiteralPath $required)) {
        throw "[semantic-labeling-governance] missing required file: $required"
    }
}

function Get-MatrixRegex {
    param(
        [Parameter(Mandatory = $true)][string]$CheckId
    )
    $inCheck = $false
    foreach ($line in Get-Content -LiteralPath $matrixPath) {
        if ($line -match "^\s*-\s*id:\s*$([regex]::Escape($CheckId))\s*$") {
            $inCheck = $true
            continue
        }
        if ($inCheck -and $line -match "^\s*-\s*id:\s*") {
            $inCheck = $false
        }
        if ($inCheck -and $line -match '^\s*regex:\s*"(.*)"\s*$') {
            return $Matches[1].Replace('\\', '\')
        }
    }
    throw "[semantic-labeling-governance] failed to parse regex for check id: $CheckId"
}

function Get-GovernedEntries {
    $entries = New-Object 'System.Collections.Generic.List[string]'
    $inGoverned = $false
    foreach ($line in Get-Content -LiteralPath $matrixPath) {
        if (-not $inGoverned -and $line -match '^\s*governed:\s*$') {
            $inGoverned = $true
            continue
        }
        if ($inGoverned -and $line -match '^\s*checks:\s*$') {
            break
        }
        if ($inGoverned -and $line -match '^\s*-\s*(.+?)\s*$') {
            $entries.Add($Matches[1].Trim()) | Out-Null
        }
    }
    if ($entries.Count -eq 0) {
        throw "[semantic-labeling-governance] governed scope is empty in matrix"
    }
    return $entries
}

function Add-UniqueString {
    param(
        [Parameter(Mandatory = $true)][AllowEmptyCollection()][System.Collections.Generic.HashSet[string]]$Set,
        [Parameter(Mandatory = $true)][string]$Value
    )
    if ([string]::IsNullOrWhiteSpace($Value)) {
        return
    }
    [void]$Set.Add($Value.Trim())
}

function Normalize-KeyPath {
    param(
        [Parameter(Mandatory = $true)][string]$Path
    )
    return $Path.Replace('\', '/')
}

function Add-Count {
    param(
        [Parameter(Mandatory = $true)][hashtable]$CurrentCounts,
        [Parameter(Mandatory = $true)][hashtable]$Totals,
        [Parameter(Mandatory = $true)][string]$Rule,
        [Parameter(Mandatory = $true)][string]$Path
    )
    $normalizedPath = Normalize-KeyPath -Path $Path
    $key = "$Rule|$normalizedPath"
    if ($CurrentCounts.ContainsKey($key)) {
        $CurrentCounts[$key] = [int]$CurrentCounts[$key] + 1
    }
    else {
        $CurrentCounts[$key] = 1
    }

    if ($Totals.ContainsKey($Rule)) {
        $Totals[$Rule] = [int]$Totals[$Rule] + 1
    }
    else {
        $Totals[$Rule] = 1
    }
}

function Collect-ContentRuleCounts {
    param(
        [Parameter(Mandatory = $true)][string]$Rule,
        [Parameter(Mandatory = $true)][string]$Regex,
        [Parameter(Mandatory = $true)][string[]]$ScanTargets,
        [Parameter(Mandatory = $true)][hashtable]$CurrentCounts,
        [Parameter(Mandatory = $true)][hashtable]$Totals
    )
    if ($ScanTargets.Count -eq 0) {
        return
    }
    $hits = @(Invoke-NativeCaptureStrict -AllowFailure -Label "rg -n -- $Rule" -Command {
            rg -n -- $Regex @ScanTargets
        })
    foreach ($hit in $hits) {
        $line = [string]$hit
        if ([string]::IsNullOrWhiteSpace($line)) {
            continue
        }
        $parts = $line.Split(":", 3)
        if ($parts.Count -lt 2) {
            continue
        }
        $path = $parts[0].Trim()
        if ([string]::IsNullOrWhiteSpace($path)) {
            continue
        }
        Add-Count -CurrentCounts $CurrentCounts -Totals $Totals -Rule $Rule -Path $path
    }
}

function Collect-PathRuleCounts {
    param(
        [Parameter(Mandatory = $true)][string]$Rule,
        [Parameter(Mandatory = $true)][string]$Regex,
        [Parameter(Mandatory = $true)][string[]]$GovernedFiles,
        [Parameter(Mandatory = $true)][hashtable]$CurrentCounts,
        [Parameter(Mandatory = $true)][hashtable]$Totals
    )
    if ($GovernedFiles.Count -eq 0) {
        return
    }
    $hits = @(Invoke-NativeCaptureStrict -AllowFailure -Label "rg -n -- $Rule (path scan)" -Command {
            $GovernedFiles | rg -n -- $Regex
        })
    foreach ($hit in $hits) {
        $line = [string]$hit
        if ([string]::IsNullOrWhiteSpace($line)) {
            continue
        }
        $parts = $line.Split(":", 2)
        if ($parts.Count -lt 2) {
            continue
        }
        $path = $parts[1].Trim()
        if ([string]::IsNullOrWhiteSpace($path)) {
            continue
        }
        Add-Count -CurrentCounts $CurrentCounts -Totals $Totals -Rule $Rule -Path $path
    }
}

$governedEntries = @(Get-GovernedEntries)
$scanTargetSet = New-Object 'System.Collections.Generic.HashSet[string]'
$governedFileSet = New-Object 'System.Collections.Generic.HashSet[string]'

foreach ($entry in $governedEntries) {
    if ($entry.EndsWith("/**", [System.StringComparison]::Ordinal)) {
        $prefix = $entry.Substring(0, $entry.Length - 3)
        if (Test-Path -LiteralPath $prefix) {
            Add-UniqueString -Set $scanTargetSet -Value $prefix
        }
        $files = @(Invoke-NativeCaptureStrict -AllowFailure -Label "git ls-files -- $prefix" -Command {
                git ls-files -- $prefix
            })
        foreach ($file in $files) {
            Add-UniqueString -Set $governedFileSet -Value ([string]$file)
        }
        continue
    }

    if (Test-Path -LiteralPath $entry) {
        Add-UniqueString -Set $scanTargetSet -Value $entry
    }
    $files = @(Invoke-NativeCaptureStrict -AllowFailure -Label "git ls-files -- $entry" -Command {
            git ls-files -- $entry
        })
    foreach ($file in $files) {
        Add-UniqueString -Set $governedFileSet -Value ([string]$file)
    }
}

$scanTargets = @($scanTargetSet)
$governedFiles = @($governedFileSet)
if ($scanTargets.Count -eq 0) {
    throw "[semantic-labeling-governance] no scan targets resolved from matrix"
}
if ($governedFiles.Count -eq 0) {
    throw "[semantic-labeling-governance] no governed files resolved from matrix"
}

$axxContentRegex = Get-MatrixRegex -CheckId "legacy-axx-content"
$axxPathRegex = Get-MatrixRegex -CheckId "legacy-axx-path"
$caRegex = Get-MatrixRegex -CheckId "legacy-context-stage-wording"

$currentCounts = @{}
$totals = @{}
Collect-ContentRuleCounts -Rule "legacy-axx-content" -Regex $axxContentRegex -ScanTargets $scanTargets -CurrentCounts $currentCounts -Totals $totals
Collect-ContentRuleCounts -Rule "legacy-context-stage-wording-content" -Regex $caRegex -ScanTargets $scanTargets -CurrentCounts $currentCounts -Totals $totals
Collect-PathRuleCounts -Rule "legacy-axx-path" -Regex $axxPathRegex -GovernedFiles $governedFiles -CurrentCounts $currentCounts -Totals $totals
Collect-PathRuleCounts -Rule "legacy-context-stage-wording-path" -Regex $caRegex -GovernedFiles $governedFiles -CurrentCounts $currentCounts -Totals $totals

$baselineRows = @(Import-Csv -LiteralPath $baselinePath)
$baselineCounts = @{}
foreach ($row in $baselineRows) {
    $rule = if ($null -eq $row.rule) { "" } else { [string]$row.rule }
    $path = if ($null -eq $row.path) { "" } else { Normalize-KeyPath -Path ([string]$row.path) }
    $baselineRaw = if ($null -eq $row.baseline_count) { "" } else { [string]$row.baseline_count }
    if ([string]::IsNullOrWhiteSpace($rule) -or [string]::IsNullOrWhiteSpace($path)) {
        continue
    }
    $baseline = 0
    if (-not [int]::TryParse($baselineRaw, [ref]$baseline) -or $baseline -lt 0) {
        throw "[semantic-labeling-governance] invalid baseline count in $baselinePath : $rule,$path,$baselineRaw"
    }
    $baselineCounts["$rule|$path"] = $baseline
}

$violations = New-Object 'System.Collections.Generic.List[string]'
foreach ($key in $currentCounts.Keys) {
    $current = [int]$currentCounts[$key]
    if (-not $baselineCounts.ContainsKey($key)) {
        $violations.Add("[semantic-labeling-governance][violation] new naming debt path detected: $key current=$current") | Out-Null
        continue
    }
    $baseline = [int]$baselineCounts[$key]
    if ($current -gt $baseline) {
        $violations.Add("[semantic-labeling-governance][violation] naming debt expanded: $key current=$current baseline=$baseline") | Out-Null
    }
}

$mappingDuplicates = @(Invoke-NativeCaptureStrict -AllowFailure -Label "rg duplicate mapping definitions" -Command {
        rg -n -- '^\s*(legacy_aliases|context_assembler_stage_mapping):\s*$' @scanTargets
    })
if ($mappingDuplicates.Count -gt 0) {
    $violations.Add("[semantic-labeling-governance][violation] duplicate mapping definitions found outside canonical source:") | Out-Null
    foreach ($line in $mappingDuplicates) {
        $text = [string]$line
        if (-not [string]::IsNullOrWhiteSpace($text)) {
            $violations.Add("  $text") | Out-Null
        }
    }
}

$legacyAxxContentTotal = if ($totals.ContainsKey("legacy-axx-content")) { [int]$totals["legacy-axx-content"] } else { 0 }
$legacyContextContentTotal = if ($totals.ContainsKey("legacy-context-stage-wording-content")) { [int]$totals["legacy-context-stage-wording-content"] } else { 0 }
$legacyAxxPathTotal = if ($totals.ContainsKey("legacy-axx-path")) { [int]$totals["legacy-axx-path"] } else { 0 }
$legacyContextPathTotal = if ($totals.ContainsKey("legacy-context-stage-wording-path")) { [int]$totals["legacy-context-stage-wording-path"] } else { 0 }

Write-Host "[semantic-labeling-governance] summary:"
Write-Host "  legacy-axx-content=$legacyAxxContentTotal"
Write-Host "  legacy-context-stage-wording-content=$legacyContextContentTotal"
Write-Host "  legacy-axx-path=$legacyAxxPathTotal"
Write-Host "  legacy-context-stage-wording-path=$legacyContextPathTotal"
Write-Host "  baseline_rows=$($baselineRows.Count)"

if ($violations.Count -gt 0) {
    foreach ($line in $violations) {
        Write-Host $line
    }
    throw "[semantic-labeling-governance] failed: violations=$($violations.Count)"
}

Write-Host "[semantic-labeling-governance] passed"
