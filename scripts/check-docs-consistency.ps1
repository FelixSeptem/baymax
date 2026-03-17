Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $repoRoot

$readme = Get-Content -Raw README.md
$docMatches = [regex]::Matches($readme, "docs/[A-Za-z0-9\-]+\.md")

$missing = @()
foreach ($m in $docMatches) {
    $path = $m.Value
    if (-not (Test-Path $path)) {
        $missing += $path
    }
}

if ($missing.Count -gt 0) {
    Write-Error ("Missing docs references in README: " + ($missing -join ", "))
}

$cfgDoc = Get-Content -Raw docs/runtime-config-diagnostics.md
if ($cfgDoc -notmatch "迁移映射") {
    Write-Error "docs/runtime-config-diagnostics.md must include migration mapping section."
}

$boundaryDoc = Get-Content -Raw docs/runtime-module-boundaries.md
if ($boundaryDoc -notmatch "依赖方向") {
    Write-Error "docs/runtime-module-boundaries.md must include dependency direction section."
}

go test ./tool/contributioncheck -run '^TestMainlineContractIndexReferencesExistingTests$' -count=1

Write-Host "Docs consistency check passed."
