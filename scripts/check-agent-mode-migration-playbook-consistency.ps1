Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$matrixPath = Join-Path $repoRoot "examples/agent-modes/MATRIX.md"
$playbookPath = Join-Path $repoRoot "examples/agent-modes/PLAYBOOK.md"

Write-Host "[agent-mode-migration-playbook-consistency] validating matrix/playbook/readme consistency"

if (-not (Test-Path -LiteralPath $matrixPath -PathType Leaf)) {
    throw "[agent-mode-migration-playbook-consistency][missing-checklist] missing matrix: $matrixPath"
}
if (-not (Test-Path -LiteralPath $playbookPath -PathType Leaf)) {
    throw "[agent-mode-migration-playbook-consistency][missing-checklist] missing playbook: $playbookPath"
}

$matrixLines = Get-Content -Path $matrixPath
$playbookRaw = Get-Content -Path $playbookPath -Raw

$missingChecklist = New-Object 'System.Collections.Generic.List[string]'
$missingGate = New-Object 'System.Collections.Generic.List[string]'

foreach ($line in $matrixLines) {
    if ($line -notmatch '^\| `[^`]+` \|') {
        continue
    }
    $parts = $line.Split('|')
    if ($parts.Count -lt 7) {
        continue
    }

    $pattern = $parts[1].Trim().Trim('`')
    $gatesCell = $parts[5].Trim()
    $readmePath = Join-Path $repoRoot "examples/agent-modes/$pattern/production-ish/README.md"
    $readmeRaw = ""
    if (Test-Path -LiteralPath $readmePath -PathType Leaf) {
        $readmeRaw = Get-Content -Path $readmePath -Raw
        if (-not $readmeRaw.Contains("## Prod Delta Checklist")) {
            $missingChecklist.Add("$pattern:missing-prod-delta-checklist") | Out-Null
        }
    }
    else {
        $missingChecklist.Add("$pattern:missing-production-ish-readme") | Out-Null
    }

    $patternToken = ('`{0}`' -f $pattern)
    if (-not $playbookRaw.Contains($patternToken)) {
        $missingChecklist.Add("$pattern:missing-playbook-pattern-mapping") | Out-Null
    }

    $gateMatches = [Regex]::Matches($gatesCell, '`([^`]+)`')
    foreach ($match in $gateMatches) {
        $gate = $match.Groups[1].Value
        if ([string]::IsNullOrWhiteSpace($gate)) {
            continue
        }
        if (-not $playbookRaw.Contains($gate)) {
            $missingGate.Add("$pattern:playbook-missing-gate:$gate") | Out-Null
        }
        if (-not [string]::IsNullOrWhiteSpace($readmeRaw) -and -not $readmeRaw.Contains($gate)) {
            $missingGate.Add("$pattern:production-ish-missing-gate:$gate") | Out-Null
        }
    }
}

if ($missingChecklist.Count -gt 0) {
    Write-Host "[agent-mode-migration-playbook-consistency][missing-checklist] inconsistencies found:"
    foreach ($item in $missingChecklist) {
        Write-Host "  - $item"
    }
}
if ($missingGate.Count -gt 0) {
    Write-Host "[agent-mode-migration-playbook-consistency][missing-gate] inconsistencies found:"
    foreach ($item in $missingGate) {
        Write-Host "  - $item"
    }
}

if ($missingChecklist.Count -gt 0 -or $missingGate.Count -gt 0) {
    exit 1
}

Write-Host "[agent-mode-migration-playbook-consistency] consistency is complete"
