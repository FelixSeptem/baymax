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

$requiredSections = @(
    "## Run",
    "## Prerequisites",
    "## Real Runtime Path",
    "## Expected Output/Verification",
    "## Failure/Rollback Notes"
)

$matrixLines = Get-Content -Path $matrixPath
$playbookRaw = Get-Content -Path $playbookPath -Raw

$missingChecklist = New-Object 'System.Collections.Generic.List[string]'
$missingGate = New-Object 'System.Collections.Generic.List[string]'

$headerLine = $matrixLines | Where-Object { $_ -match '^\|\s*pattern\s*\|' } | Select-Object -First 1
if ([string]::IsNullOrWhiteSpace($headerLine)) {
    throw "[agent-mode-migration-playbook-consistency][missing-checklist] missing matrix header row: $matrixPath"
}
$headerParts = $headerLine.Split('|') | ForEach-Object { $_.Trim() }
$patternColumnIndex = [Array]::IndexOf($headerParts, 'pattern')
$gatesColumnIndex = [Array]::IndexOf($headerParts, 'gates')
if ($patternColumnIndex -lt 0 -or $gatesColumnIndex -lt 0) {
    throw "[agent-mode-migration-playbook-consistency][missing-checklist] missing required columns in matrix header (pattern/gates)"
}

if (-not $playbookRaw.Contains("## Production Migration Checklist")) {
    $missingChecklist.Add("playbook:missing-production-migration-checklist") | Out-Null
}

foreach ($line in $matrixLines) {
    if ($line -notmatch '^\| `[^`]+` \|') {
        continue
    }
    $parts = $line.Split('|')
    if ($parts.Count -le [Math]::Max($patternColumnIndex, $gatesColumnIndex)) {
        continue
    }

    $pattern = $parts[$patternColumnIndex].Trim().Trim('`')
    if ([string]::IsNullOrWhiteSpace($pattern)) {
        continue
    }
    $gatesCell = $parts[$gatesColumnIndex].Trim().Trim('`')
    $readmePath = Join-Path $repoRoot "examples/agent-modes/$pattern/production-ish/README.md"
    $readmeRaw = ""

    if (-not (Test-Path -LiteralPath $readmePath -PathType Leaf)) {
        $missingChecklist.Add("${pattern}:missing-production-ish-readme") | Out-Null
    }
    else {
        $readmeRaw = Get-Content -Path $readmePath -Raw
        foreach ($section in $requiredSections) {
            if (-not $readmeRaw.Contains($section)) {
                $missingChecklist.Add("${pattern}:missing-section:${section}") | Out-Null
            }
        }
    }

    $patternToken = ('`{0}`' -f $pattern)
    if (-not $playbookRaw.Contains($patternToken)) {
        $missingChecklist.Add("${pattern}:missing-playbook-pattern-mapping") | Out-Null
    }

    $gateTokens = $gatesCell.Split(';')
    foreach ($rawGate in $gateTokens) {
        $gate = $rawGate.Trim()
        if ([string]::IsNullOrWhiteSpace($gate) -or $gate -eq "-") {
            continue
        }
        if (-not $playbookRaw.Contains($gate)) {
            $missingGate.Add("${pattern}:playbook-missing-gate:${gate}") | Out-Null
        }
        if (-not [string]::IsNullOrWhiteSpace($readmeRaw) -and -not $readmeRaw.Contains($gate)) {
            $missingGate.Add("${pattern}:production-ish-missing-gate:${gate}") | Out-Null
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
