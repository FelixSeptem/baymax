Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

Write-Host "[agent-mode-legacy-todo-cleanup] scanning examples for unresolved placeholders"

$rg = Get-Command rg -ErrorAction SilentlyContinue
if ($null -eq $rg) {
    throw "[agent-mode-legacy-todo-cleanup] ripgrep (rg) is required"
}

$matchesRaw = & rg -n "TODO|TBD|FIXME|待补" examples
$status = $LASTEXITCODE

if ($status -eq 0) {
    $filteredMatches = @()
    foreach ($line in $matchesRaw) {
        if ($line -match 'examples[\\/]+agent-modes[\\/]+LEGACY_TODO_BASELINE.md') {
            continue
        }
        $filteredMatches += $line
    }
    if ($filteredMatches.Count -gt 0) {
        Write-Host "[agent-mode-legacy-todo-cleanup][legacy-placeholder] unresolved placeholders found:"
        foreach ($line in $filteredMatches) {
            Write-Host $line
        }
        exit 1
    }
    Write-Host "[agent-mode-legacy-todo-cleanup] cleanup is complete"
    exit 0
}
if ($status -ne 1) {
    throw "[agent-mode-legacy-todo-cleanup] placeholder scan failed"
}

Write-Host "[agent-mode-legacy-todo-cleanup] cleanup is complete"
