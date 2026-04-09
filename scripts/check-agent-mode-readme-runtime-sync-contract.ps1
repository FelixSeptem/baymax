Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[agent-mode-readme-runtime-sync-contract] validating README runtime sync for agent-mode examples"

$mainFiles = Get-ChildItem -Path "examples/agent-modes" -Recurse -Filter main.go -File |
Where-Object { $_.FullName -notmatch "\\internal\\" } |
Sort-Object -Property FullName

if ($mainFiles.Count -eq 0) {
    throw "[agent-mode-readme-runtime-sync-contract] no agent-mode main.go found"
}

$requiredSections = @(
    "## Run",
    "## Prerequisites",
    "## Real Runtime Path",
    "## Expected Output/Verification",
    "## Failure/Rollback Notes"
)

$missingRequiredSections = New-Object 'System.Collections.Generic.List[string]'

foreach ($mainFile in $mainFiles) {
    $readmePath = Join-Path $mainFile.DirectoryName "README.md"
    if (-not (Test-Path -LiteralPath $readmePath -PathType Leaf)) {
        $missingRequiredSections.Add("$readmePath:missing-readme") | Out-Null
        continue
    }
    $raw = Get-Content -Path $readmePath -Raw
    foreach ($section in $requiredSections) {
        if (-not $raw.Contains($section)) {
            $missingRequiredSections.Add("$readmePath:missing-section:$section") | Out-Null
        }
    }
}

$statusLines = @(git status --porcelain -- examples/agent-modes)
$changedMainFiles = New-Object 'System.Collections.Generic.List[string]'
$changedReadmeFiles = New-Object 'System.Collections.Generic.List[string]'
foreach ($line in $statusLines) {
    $trimmed = $line.Trim()
    if ([string]::IsNullOrWhiteSpace($trimmed)) {
        continue
    }
    $parts = $trimmed -split '\s+'
    if ($parts.Count -eq 0) {
        continue
    }
    $path = $parts[$parts.Count - 1]
    if ($path -like "examples/agent-modes/*/main.go") {
        $changedMainFiles.Add($path) | Out-Null
    }
    if ($path -like "examples/agent-modes/*/README.md") {
        $changedReadmeFiles.Add($path) | Out-Null
    }
}

$readmeNotUpdated = New-Object 'System.Collections.Generic.List[string]'
foreach ($mainPath in $changedMainFiles) {
    $readmePath = (Join-Path (Split-Path -Path $mainPath -Parent) "README.md").Replace("\", "/")
    $matched = $false
    foreach ($changedReadme in $changedReadmeFiles) {
        if ($changedReadme.Replace("\", "/") -eq $readmePath) {
            $matched = $true
            break
        }
    }
    if (-not $matched) {
        $readmeNotUpdated.Add("$mainPath -> $readmePath") | Out-Null
    }
}

if ($readmeNotUpdated.Count -gt 0) {
    Write-Host "[agent-mode-readme-runtime-sync-contract][agent-mode-readme-runtime-desync] main.go changed without matching README update:"
    foreach ($item in $readmeNotUpdated) {
        Write-Host "  - $item"
    }
}

if ($missingRequiredSections.Count -gt 0) {
    Write-Host "[agent-mode-readme-runtime-sync-contract][agent-mode-readme-required-sections-missing] required README sections missing:"
    foreach ($item in $missingRequiredSections) {
        Write-Host "  - $item"
    }
}

if ($readmeNotUpdated.Count -gt 0 -or $missingRequiredSections.Count -gt 0) {
    exit 1
}

Write-Host "[agent-mode-readme-runtime-sync-contract] passed"