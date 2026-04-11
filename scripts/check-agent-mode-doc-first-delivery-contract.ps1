Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[agent-mode-doc-first-delivery-contract] validating doc-first delivery constraints"

$matrixFile = Join-Path $repoRoot "examples/agent-modes/MATRIX.md"
$playbookFile = Join-Path $repoRoot "examples/agent-modes/PLAYBOOK.md"
$baselineFile = Join-Path $repoRoot "examples/agent-modes/doc-baseline-freeze.md"

if (-not (Test-Path -LiteralPath $matrixFile -PathType Leaf)) {
    throw "[agent-mode-doc-first-delivery-contract][agent-mode-doc-first-baseline-missing] missing matrix: $matrixFile"
}
if (-not (Test-Path -LiteralPath $playbookFile -PathType Leaf)) {
    throw "[agent-mode-doc-first-delivery-contract][agent-mode-doc-first-baseline-missing] missing playbook: $playbookFile"
}
if (-not (Test-Path -LiteralPath $baselineFile -PathType Leaf)) {
    throw "[agent-mode-doc-first-delivery-contract][agent-mode-doc-first-baseline-missing] missing baseline freeze: $baselineFile"
}

$matrixRaw = Get-Content -Path $matrixFile -Raw
foreach ($token in @("doc-baseline-ready", "impl-ready", "failure_rollback_ref")) {
    if (-not $matrixRaw.Contains($token)) {
        throw "[agent-mode-doc-first-delivery-contract][agent-mode-doc-first-baseline-missing] matrix missing doc-first column: $token"
    }
}

$requiredSections = @(
    "## Run",
    "## Prerequisites",
    "## Real Runtime Path",
    "## Expected Output/Verification",
    "## Failure/Rollback Notes"
)

$missingReadmeSections = New-Object 'System.Collections.Generic.List[string]'
$readmes = Get-ChildItem -Path "examples/agent-modes" -Recurse -Filter README.md -File |
Where-Object { $_.FullName -notmatch "\\internal\\" -and $_.FullName -notmatch "\\.tmp\\" } |
Sort-Object -Property FullName

foreach ($readme in $readmes) {
    $raw = Get-Content -Path $readme.FullName -Raw
    foreach ($section in $requiredSections) {
        if (-not $raw.Contains($section)) {
            $missingReadmeSections.Add("$($readme.FullName):missing-section:$section") | Out-Null
        }
    }
    if ($readme.FullName -match "\\production-ish\\README.md$") {
        if (-not $raw.Contains("## Variant Delta (vs minimal)")) {
            $missingReadmeSections.Add("$($readme.FullName):missing-section:## Variant Delta (vs minimal)") | Out-Null
        }
    }
}

$statusLines = @(git status --porcelain -- examples/agent-modes)
$changedCodeFiles = New-Object 'System.Collections.Generic.List[string]'
foreach ($line in $statusLines) {
    $trimmed = ([string]$line).Trim()
    if ([string]::IsNullOrWhiteSpace($trimmed)) {
        continue
    }
    $parts = $trimmed -split '\s+'
    if ($parts.Count -eq 0) {
        continue
    }
    $path = $parts[$parts.Count - 1].Replace("\", "/")
    if ($path -match '^examples/agent-modes/[^/]+/semantic_example\.go$' -or
        $path -match '^examples/agent-modes/[^/]+/(minimal|production-ish)/main\.go$') {
        $changedCodeFiles.Add($path) | Out-Null
    }
}

$docFirstBaselineMissing = New-Object 'System.Collections.Generic.List[string]'
if ($changedCodeFiles.Count -gt 0) {
    $matrixLines = Get-Content -Path $matrixFile
    foreach ($codePath in $changedCodeFiles) {
        $segments = $codePath.Split("/")
        if ($segments.Count -lt 4) {
            $docFirstBaselineMissing.Add("${codePath}:invalid-path-shape") | Out-Null
            continue
        }
        $pattern = $segments[2]
        $rowToken = "| ``$pattern`` |"
        $row = ($matrixLines | Where-Object { $_.Contains($rowToken) } | Select-Object -First 1)
        if ([string]::IsNullOrWhiteSpace($row)) {
            $docFirstBaselineMissing.Add("${codePath}:missing-matrix-row") | Out-Null
            continue
        }
        if ($row -notmatch '\|\s*yes\s*\|\s*yes\s*\|') {
            $docFirstBaselineMissing.Add("${codePath}:doc-baseline-ready/impl-ready not both yes") | Out-Null
        }
    }
}

if ($docFirstBaselineMissing.Count -gt 0) {
    Write-Host "[agent-mode-doc-first-delivery-contract][agent-mode-doc-first-baseline-missing] doc-first baseline is incomplete for changed code paths:"
    foreach ($item in $docFirstBaselineMissing) {
        Write-Host "  - $item"
    }
}

if ($missingReadmeSections.Count -gt 0) {
    Write-Host "[agent-mode-doc-first-delivery-contract][agent-mode-doc-required-sections-missing] required doc sections missing:"
    foreach ($item in $missingReadmeSections) {
        Write-Host "  - $item"
    }
}

if ($docFirstBaselineMissing.Count -gt 0 -or $missingReadmeSections.Count -gt 0) {
    exit 1
}

Write-Host "[agent-mode-doc-first-delivery-contract] passed"
